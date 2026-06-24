# Intake: Extração do Konvario para serviço independente

**Origin:** Solicitação do usuário — extrair a funcionalidade de "resource tenant" hoje embutida no `front-manager-api` (Spring Boot/Java) para um serviço autônomo em `djalmajr/konvario/`.

## Context

- **Problem/opportunity:** A gestão de "resource tenant" (configurações de recursos por tenant + aplicação) vive acoplada ao `front-manager-api`. A lógica de manutenção/distribuição de cache está numa lib compartilhada (`hyper-resource-tenant-lib`) e a API de escrita/leitura está no monólito. Isso dificulta evolução isolada, deploy independente e reuso por outros serviços da plataforma que já consomem esses dados.
- **Initial objective:** Criar um serviço dedicado (`Konvario`) que seja dono do domínio de configuração de recursos por tenant — incluindo persistência, criptografia de campos sensíveis e distribuição via mensageria/cache — mantendo **100% de compatibilidade comportamental** com a implementação atual, pois já existem serviços consumindo esses dados.
- **Expected value signal:** Deploy e evolução independentes do domínio; redução de acoplamento do `front-manager-api`; base para reuso multi-aplicação. (Sem KPIs de produto — valor é arquitetural/operacional.)
- **Constraints and assumptions:**
  - **Compatibilidade total** com o formato de dados atual (banco, payload Kafka, formato do cache em object storage) — consumidores existentes não podem quebrar.
  - **Criptografia idêntica à atual** no port (ver nota de risco abaixo). Não corrigir o algoritmo agora; apenas portar fielmente.
  - Port do `SecurityGenerator` já existe e está fiel: `run2biz/edge-runtime/server/libs/security-generator.ts` (DES + PBE estilo `PBEWithMD5AndDES`, 20 iterações, senha embutida).
  - Stack candidata: **Bun** (preferência inicial) — avaliar **Go** como alternativa (tabela comparativa abaixo).
  - **Fora de escopo neste momento:** adoção de cofre de segredos externo (decisão adiada deliberadamente).

## Initial scope

### Includes
- **Modelo de dados** (PostgreSQL), portando as entidades:
  - `resource_template` — recurso suportado pela plataforma (nome, descrição, versão, `type`, `status`) + seus campos.
  - `resource_field_setting` — definição dinâmica de campo (`label`, `message_key`, `hint`, `required`, `sensitive_data`).
  - `resource_tenant` — associação tenant + aplicação + template + status.
  - `resource_tenant_field_value` — valores dos campos por resource tenant (sensíveis criptografados).
- **API REST** equivalente à atual (`/internal/resource-tenant`):
  - `GET /platform-application/{identificationKey}` — ativos por aplicação.
  - `GET /platform-application/{identificationKey}/domain/{domain}` — ativos por domínio.
  - `GET /platform-application/{identificationKey}/tenant/{tenant}` (+ filtro opcional `resourceType`).
  - `POST /platform-application/{identificationKey}/tenant/{tenant}` — criação.
  - `PUT .../resource-tenant-id/{id}/disable` — desativação.
- **Criptografia de campos sensíveis** (`sensitive_data = true`) no create, com algoritmo idêntico ao atual (reuso do port `security-generator.ts`).
- **Distribuição via Kafka** — publicação do `ResourceTenantBody` em create/disable (equivalente a `createBlockSend` / `disableBlockSend`), mantendo o mesmo contrato de mensagem.
- **Manutenção de cache** em object storage (MinIO, bucket `config-application-caches`) — equivalente ao `@EnableResourceTenantCacheMaintainer` da lib.
- **Regras de negócio** atuais: validação na criação, `verifyContainsSomeEnabled` (impede dois recursos ativos para mesmo tenant+template+app).
- **Decisão de stack** Bun vs Go documentada.

### Does not include
- Adoção de Vault/OpenBao ou qualquer cofre externo.
- Refatoração/correção do algoritmo de criptografia.
- Mudança de contrato com consumidores existentes.
- Migração dos demais domínios do `front-manager-api`.

## Inputs and references

- **Stakeholders:** time de plataforma (donos do `front-manager-api` e dos serviços consumidores do resource tenant).
- **Documents/links:**
  - Implementação atual: `run2biz/front-manager-api/src/main/java/.../service/impl/ResourceTenantServiceImpl.java`
  - Controller: `.../controller/ResourceTenantResource.java`
  - Entidades: `.../domain/jpa/ResourceTenant.java`, `ResourceTemplate.java`, `ResourceFieldSetting.java`, `ResourceTenantFieldValue.java`
  - Lib de distribuição/cache: `hyper-resource-tenant-lib` (`@EnableResourceTenantCacheMaintainer`, `ResourceTenantKafkaService`, `ResourceTenantBody`)
  - Criptografia original: `.../security/util/ObfuscateUtil.java`
  - Port TS já existente: `run2biz/edge-runtime/server/libs/security-generator.ts`
- **Known technical context:**
  - Dependências de infra: **PostgreSQL** (dados), **Kafka** (distribuição, tópico `config-application-caches`), **MinIO** (cache em object storage).
  - Endpoints são `/internal/*` — comunicação serviço-a-serviço (avaliar modelo de auth no novo serviço).
  - A semântica exata de **formato do cache** e do **payload Kafka** vive dentro da lib `hyper-resource-tenant-lib` (não neste repo) — precisa ser engenharia reversa / inspeção para garantir compatibilidade.

## Nota de risco — criptografia atual (registro, sem ação agora)

> **Não corrigir nesta extração.** O port deve replicar o comportamento para manter compatibilidade com dados já persistidos por consumidores existentes. Registro apenas para visibilidade futura.

O algoritmo atual (`ObfuscateUtil` / `SecurityGenerator`) apresenta fragilidades conhecidas:
- **DES** (chave efetiva de 56 bits) — cifra obsoleta e quebrável.
- **Derivação de chave fraca:** `PBEWithMD5AndDES` com apenas **20 iterações** e MD5.
- **Senha embutida no código-fonte** (`PASSWORD = "jtr0>eOwb!)8)k!E"`), versionada e replicada em múltiplos repositórios.
- Sem rotação de chave, sem autenticação da cifra (sem AEAD), prefixo fixo `!@,`.

Recomendação futura (item separado, pós-extração): migrar para AEAD moderno (ex.: AES-256-GCM) com chave externalizada e estratégia de rotação/recriptografia dos valores existentes — tratado como iniciativa própria, fora deste intake.

## Decisão de stack — Bun vs Go (comparativo)

| Critério | Bun (TypeScript) | Go |
|---|---|---|
| **Reuso do port de cripto** | ✅ `security-generator.ts` já existe e fiel — reuso imediato, mesmo ecossistema TS do `edge-runtime` | ⚠️ Reimplementar DES/PBE em Go (viável via `crypto/des`, mas é novo port a validar byte-a-byte) |
| **Compatibilidade DES/PBE legado** | ✅ `crypto-js` já reproduz o esquema atual | ✅ stdlib `crypto/des` + derivação manual; exige cuidado para bater com o legado |
| **Performance / footprint** | Bom; runtime único, startup rápido | ✅ Melhor uso de CPU/memória, binário único estático, ótimo p/ serviços de I/O concorrente |
| **Concorrência** | Event loop assíncrono; suficiente p/ I/O-bound | ✅ Goroutines — modelo superior p/ consumo Kafka + workers de cache em paralelo |
| **Ecossistema Kafka** | `kafkajs` (maduro) ou cliente nativo | ✅ `franz-go` / `confluent-kafka-go` — clientes robustos e amplamente usados |
| **Ecossistema PostgreSQL** | `postgres`/`drizzle`/`prisma` | `pgx` (excelente) |
| **Ecossistema MinIO/S3** | SDK S3 (`@aws-sdk` ou nativo) | ✅ SDK MinIO oficial em Go é first-class |
| **Velocidade de desenvolvimento** | ✅ Alta — TS, tipos compartilháveis com outros serviços TS da casa | Média — mais verboso, porém explícito |
| **Consistência com a casa** | ✅ Alinha com `edge-runtime` (TS) | ⚠️ Introduz/《amplia》 stack Go na plataforma |
| **Maturidade do runtime** | Bun ainda jovem (mudanças rápidas) | ✅ Go muito maduro e estável p/ backend |
| **Tipagem / robustez** | TS (tipagem estrutural, opt-in em runtime) | ✅ Tipagem forte compilada |
| **Deploy / distribuição** | Imagem com runtime Bun | ✅ Binário estático mínimo, imagens enxutas |

**Síntese:**
- **Bun** vence em *time-to-market* e reuso direto do port de criptografia já validado — menor risco de divergência de comportamento na parte mais sensível (compatibilidade da cifra). Forte se a prioridade é entregar rápido e manter consistência com o `edge-runtime`.
- **Go** vence em robustez operacional, concorrência (consumo Kafka + manutenção de cache) e ecossistema maduro de infra (Kafka/MinIO/Postgres). Forte se o serviço tende a alto volume e longevidade, e se Go já for/for adotado como padrão de backend.

Recomendação para discussão no epic: **se o reuso imediato do `security-generator.ts` e a velocidade forem decisivos → Bun**; **se robustez de longo prazo e concorrência pesada forem decisivos → Go** (assumindo o custo de portar e validar a cripto byte-a-byte). Decisão final fica como questão aberta para o epic.

## Open questions

- [ ] **Stack definitiva: Bun ou Go?** (ver comparativo — decisão de arquitetura para o epic)
- [ ] Qual o **formato exato do payload Kafka** e do **arquivo de cache no MinIO** produzidos pela `hyper-resource-tenant-lib`? (necessário inspecionar a lib para port fiel)
- [ ] Modelo de **autenticação/autorização** do novo serviço para os endpoints `/internal/*` (mantém OAuth2/Keycloak? mTLS? token de serviço?).
- [ ] Estratégia de **dados**: novo serviço compartilha o mesmo schema/banco do `front-manager-api` ou ganha banco próprio com migração? Como fica a transição dos consumidores?
- [ ] O `front-manager-api` passa a **chamar o novo serviço** ou mantém leitura própria durante transição (strangler)?
- [ ] Há outros **consumidores** além dos conhecidos que dependem do contrato atual (Kafka/cache)? Inventário necessário.
- [ ] Enum `ResourceType` e seus valores suportados — confirmar lista completa para o port.

## Recommended next step

`/agile-epic` — A extração é uma **iniciativa coordenada com múltiplas dependências** (compatibilidade de dados, contrato Kafka/cache, criptografia legada, consumidores existentes, decisão de stack e possível migração de banco). Justifica decomposição em stories: definição de stack, modelo de dados/migração, port da criptografia, API REST, integração Kafka, manutenção de cache, estratégia de transição (strangler) e validação de compatibilidade.

## Verification

- [x] The problem is clear enough for the next step
- [x] Constraints and assumptions have been made explicit
- [x] The next artifact in the flow has been defined

<!-- Save to: planning/konvario/intake.md -->
