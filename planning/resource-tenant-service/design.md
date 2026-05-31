# Design — Resource Tenant Service

**Status:** Decisões consolidadas após sessão de grilling (2026-05-30).
**Relacionado:** [intake.md](./intake.md) · [decisions.md](./decisions.md)

> ⚠️ **Pivot em relação ao intake original.** O intake descrevia um *port fiel* do
> "resource tenant" do `front-manager-api` com 100% de compatibilidade. Essa premissa
> **caiu**. O serviço passa a ser um **serviço de configuração multi-tenant**
> independente. Os projetos `front-manager-api` e `hyper-resource-tenant-lib` passam a
> ser **inspiração de padrão**, não contrato a respeitar.

## 1. Objetivo

Serviço autônomo que é dono da **configuração de recursos por tenant**. Caso de uso central:

> Um serviço consumidor identifica o tenant pela **URL/hostname** → consulta o
> `resource-tenant` → recebe a **conexão de banco daquele tenant** (cada tenant tem
> banco próprio), além de **bucket MinIO** e outras configurações por tenant.

Valor: multi-tenancy real (banco/storage/config isolados por cliente), descoberta de
configuração centralizada, evolução e deploy independentes.

## 2. Visão de arquitetura

- Serviço **independente**, com **banco próprio** (tabelas próprias).
- **Stack: Go** — binário enxuto, ótimo no hot path de I/O de configuração.
  - Tooling: **chi** (router sobre net/http) + **pgx** (driver Postgres) +
    **sqlc** (SQL type-safe, sem ORM mágico) + **goose** (migrations).
- **Distribuição: REST síncrona + cache no cliente.** Sem Kafka e sem cache em
  object storage (MinIO) por ora — simplificação deliberada (eram do padrão legado
  só para compatibilidade). Pode ser reintroduzido depois se houver necessidade de
  resiliência/push.
- **Criptografia: AES-256-GCM** com chave externalizada (env/config), pronta para
  rotação. Substitui o DES/PBE/MD5 legado (quebrado, senha embutida).
- **Decrypt server-side**: o serviço descriptografa e retorna o valor em claro sobre
  TLS para consumidores autenticados. A chave de cripto fica **só** no resource-tenant.
- **Auth: API key / token de serviço + TLS** (endpoints server-to-server).

## 3. Modelo de domínio

Quatro conceitos:

- **Tenant** — o cliente, identificado por um ou mais hostnames.
- **Resource Definition** — o catálogo/schema de um *tipo* de recurso (ex: `postgres`,
  `minio`) e seus campos dinâmicos.
- **Resource Field** — definição de um campo do tipo (required, secret, etc).
- **Tenant Resource** — instância de um recurso para um tenant (status + valores).
- **Tenant Resource Value** — valor de cada campo (criptografado se secret).

### 3.1 Schema (nomes de mercado, snake_case, PostgreSQL)

```sql
-- O cliente
tenants
  id              uuid pk
  slug            text unique        -- ex: "acme" (era "alias")
  name            text
  status          text not null      -- active | inactive
  created_at      timestamptz
  updated_at      timestamptz

-- Resolução por URL (hostname -> tenant). Match exato de hostname.
-- Decisão atual: 1 hostname -> 1 tenant. Tabela própria evita migration futura
-- caso passe a N hostnames por tenant.
tenant_domains
  id              uuid pk
  tenant_id       uuid not null references tenants(id)
  hostname        text unique not null
  is_primary      bool not null default false
  created_at      timestamptz

-- Catálogo/schema do tipo de recurso (era "resource_template")
resource_definitions
  id              uuid pk
  key             text not null      -- ex: "postgres", "minio" (estável p/ consumo)
  name            text
  description     text
  version         text not null
  status          text not null      -- active | inactive
  created_at      timestamptz
  updated_at      timestamptz
  unique (key, version)

-- Campos dinâmicos do tipo (era "resource_field_setting")
resource_fields
  id              uuid pk
  resource_definition_id  uuid not null references resource_definitions(id)
  key             text not null      -- ex: "host", "password" (estável, era message_key)
  label           text
  hint            text
  data_type       text not null      -- string | int | bool | ...
  required        bool not null default false
  is_secret       bool not null default false  -- era "sensitive_data" -> dispara cripto
  sort_order      int not null default 0
  unique (resource_definition_id, key)

-- Instância de um recurso para um tenant (era "resource_tenant")
tenant_resources
  id              uuid pk
  tenant_id       uuid not null references tenants(id)
  resource_definition_id  uuid not null references resource_definitions(id)
  status          text not null      -- active | inactive
  created_at      timestamptz
  updated_at      timestamptz
  -- Invariante: no máximo 1 ativo por (tenant, definition)
  -- create unique index uq_tenant_resource_active
  --   on tenant_resources (tenant_id, resource_definition_id) where status = 'active';

-- Valor de cada campo (era "resource_tenant_field_value")
tenant_resource_values
  id              uuid pk
  tenant_resource_id  uuid not null references tenant_resources(id)
  resource_field_id   uuid not null references resource_fields(id)
  value_plain     text               -- preenchido quando NÃO secreto
  value_cipher    bytea              -- AES-256-GCM quando secreto
  nonce           bytea              -- IV do GCM (obrigatório p/ secret)
  key_version     int                -- versão da chave usada (p/ rotação futura)
  unique (tenant_resource_id, resource_field_id)

-- (Opcional, recomendado) clientes/tokens de serviço para a auth
api_clients
  id              uuid pk
  name            text
  key_hash        text not null      -- hash do token, nunca o token em claro
  status          text not null      -- active | revoked
  created_at      timestamptz
```

### 3.2 Notas de design

- **`uuid`** em vez de `bigserial` — IDs expostos em API, serviço distribuído.
- **`key` estáveis** em `resource_definitions` e `resource_fields` substituem o
  acoplamento a IDs numéricos (o legado já admitia isso via `message_key`).
- **Secret-aware no banco**: separar `value_plain` de `value_cipher`+`nonce`+`key_version`
  torna a rotação de chave viável sem mudar o contrato. AES-GCM exige guardar o nonce.
- **Invariante de unicidade** via índice único parcial (só 1 ativo por tenant+definition)
  — resolução determinística ("a" conexão do tenant).
- **Soft-delete** via `status` (espelha o `disable` legado), preserva histórico.

## 4. API

### 4.1 Consumo (server-to-server, retorna secret em claro via TLS + auth)

- `GET /v1/resolve?hostname={host}`
  → tenant + **todos** os recursos ativos (contrato "todos", bom p/ cache no cliente).
- `GET /v1/resolve/{host}/resources/{definitionKey}`
  → **um** recurso específico por tipo (atalho).

### 4.2 Admin (CRUD + validação contra `resource_definitions`)

- `/v1/tenants` (+ `/{id}/domains`)
- `/v1/resource-definitions` (+ fields)
- `/v1/tenants/{id}/resources` (+ values)

Regras:
- No create/update de `tenant_resource`, validar valores contra a definição
  (campos `required` presentes; `is_secret` dispara criptografia).
- **Mascaramento**: endpoints admin retornam valores secretos mascarados por padrão;
  `?reveal=true` (com escopo de auth adequado) revela. Endpoints de consumo retornam
  em claro.

## 5. Fora de escopo (por ora)

- Kafka / mensageria.
- Cache em object storage (MinIO) como mecanismo de distribuição.
- Cofre de segredos externo (Vault/OpenBao) — chave AES fica em env/config.
- Compatibilidade com dados/contratos do `front-manager-api`.

## 6. Decisões em aberto (não bloqueiam o epic)

- Detalhe do mascaramento admin (`?reveal=true` vs endpoint separado).
- Modelo de rotação de chave (key_version já previsto no schema).
- Tabela `api_clients` vs tokens em config — recomendado tabela (rotação/auditoria).
