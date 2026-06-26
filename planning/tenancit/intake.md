# Intake: Tenancit como serviço independente

**Origin:** necessidade recorrente de separar a configuração de recursos por tenant
de aplicações consumidoras. Este documento é uma versão sanitizada do intake
histórico; detalhes de sistemas internos e referências proprietárias foram
removidos para manter o Tenancit como produto independente.

## Context

- **Problem/opportunity:** aplicações multi-tenant frequentemente acoplam a
  configuração de recursos por tenant ao próprio backend consumidor. Isso cria
  deploys mais arriscados, dificulta reutilização e espalha detalhes de
  infraestrutura como banco, storage, autenticação e e-mail em pontos diferentes.
- **Initial objective:** criar um serviço dedicado (`Tenancit`) que seja dono do
  domínio de configuração de recursos por tenant, com persistência, criptografia
  de campos sensíveis e API de resolução por hostname.
- **Expected value signal:** deploy e evolução independentes do domínio;
  redução de acoplamento das aplicações consumidoras; base para reuso por
  múltiplos produtos e clientes.
- **Constraints and assumptions:**
  - Resolver tenant por domínio/hostname é requisito central.
  - Campos sensíveis devem ser protegidos por criptografia moderna e chave
    externalizada.
  - Tipos de recurso devem ser dinâmicos para evitar migrations a cada novo
    recurso.
  - Stack candidata inicial: **Bun**; alternativa avaliada: **Go**.
  - Integração com cofre de segredos externo fica fora do primeiro corte, mas o
    modelo deve permitir secret references no futuro.

## Initial scope

### Includes

- **Modelo de dados** (PostgreSQL):
  - `tenants` — cliente/organização atendida.
  - `tenant_domains` — hostnames que resolvem para um tenant.
  - `resource_definitions` — tipo de recurso suportado (`postgres`, `minio`,
    `smtp`, `keycloak`, etc.) e sua versão.
  - `resource_fields` — campos esperados por tipo de recurso.
  - `tenant_resources` — instância de recurso para um tenant.
  - `tenant_resource_values` — valores dos campos por recurso, com suporte a
    segredo criptografado.
- **API REST de consumo**:
  - resolver tenant e todos os recursos ativos por hostname;
  - resolver um recurso específico por hostname e tipo.
- **API REST administrativa**:
  - CRUD de tenants, domains, resource definitions, resources e API clients.
- **Criptografia de campos sensíveis** com AES-256-GCM e versão de chave.
- **Regras de negócio**:
  - validar campos obrigatórios contra a definição;
  - permitir no máximo um recurso ativo por tenant e definição;
  - mascarar valores secretos na API administrativa por padrão.
- **Decisão de stack** Bun vs Go documentada.

### Does not include

- Adoção imediata de Vault/OpenBao ou outro cofre externo.
- Mensageria/push para consumidores.
- Cache em object storage como mecanismo de distribuição.
- Compatibilidade com schemas ou contratos de implementações legadas
  específicas.

## Inputs and references

- **Stakeholders genéricos:** mantenedores de aplicações B2B multi-tenant,
  consultorias, software houses e times de plataforma que precisam gerenciar
  recursos por cliente.
- **Known technical context:**
  - Dependências de infra esperadas: PostgreSQL, API HTTP e TLS.
  - Integrações comuns: Keycloak/OIDC, MinIO/S3, SMTP, bancos dedicados e APIs
    externas.
  - Consumidores são serviços confiáveis, autenticados por API key/token de
    serviço no primeiro corte.

## Nota de risco — padrões legados

Implementações legadas desse tipo de domínio costumam conter problemas como
cifras antigas, chaves embutidas, payloads acoplados ao consumidor e caches
mantidos fora de um dono claro. O Tenancit não deve portar esses padrões. A
decisão atual é usar AES-256-GCM com chave externalizada, `key_version` no dado
criptografado e API síncrona simples no primeiro corte.

## Decisão de stack — Bun vs Go (comparativo)

| Critério | Bun (TypeScript) | Go |
|---|---|---|
| **Reuso de tipos/SDKs web** | ✅ Tipos TypeScript compartilháveis com painel e possíveis SDKs | ⚠️ Exige geração/contrato explícito para consumidores TS |
| **Criptografia moderna** | ✅ Bibliotecas maduras para AES-GCM | ✅ stdlib `crypto/aes` + `cipher.NewGCM` |
| **Performance / footprint** | Bom; runtime único, startup rápido | ✅ Melhor uso de CPU/memória, binário único estático, ótimo p/ serviços de I/O concorrente |
| **Concorrência** | Event loop assíncrono; suficiente p/ I/O-bound | ✅ Goroutines — modelo superior p/ consumo Kafka + workers de cache em paralelo |
| **Ecossistema Kafka** | `kafkajs` (maduro) ou cliente nativo | ✅ `franz-go` / `confluent-kafka-go` — clientes robustos e amplamente usados |
| **Ecossistema PostgreSQL** | `postgres`/`drizzle`/`prisma` | `pgx` (excelente) |
| **Ecossistema MinIO/S3** | SDK S3 (`@aws-sdk` ou nativo) | ✅ SDK MinIO oficial em Go é first-class |
| **Velocidade de desenvolvimento** | ✅ Alta — TS e tipos compartilháveis com o painel | Média — mais verboso, porém explícito |
| **Consistência de produto** | ✅ Uma linguagem para painel, API client e possíveis SDKs | ✅ Backend simples, binário único e operação previsível |
| **Maturidade do runtime** | Bun ainda jovem (mudanças rápidas) | ✅ Go muito maduro e estável p/ backend |
| **Tipagem / robustez** | TS (tipagem estrutural, opt-in em runtime) | ✅ Tipagem forte compilada |
| **Deploy / distribuição** | Imagem com runtime Bun | ✅ Binário estático mínimo, imagens enxutas |

**Síntese:**
- **Bun** vence em *time-to-market* e compartilhamento de tipos com painel/SDKs.
- **Go** vence em robustez operacional, concorrência (consumo Kafka + manutenção de cache) e ecossistema maduro de infra (Kafka/MinIO/Postgres). Forte se o serviço tende a alto volume e longevidade, e se Go já for/for adotado como padrão de backend.

Recomendação para discussão no epic: **se velocidade e compartilhamento de tipos
forem decisivos → Bun**; **se robustez operacional e simplicidade de deploy forem
decisivos → Go**. A decisão consolidada posterior foi Go.

## Open questions

- [ ] **Stack definitiva: Bun ou Go?** (ver comparativo — decisão de arquitetura para o epic)
- [ ] Modelo definitivo de autenticação/autorização para consumidores.
- [ ] Estratégia futura para secret references e integrações com cofre externo.
- [ ] Lista inicial de `resource_definitions` suportadas como templates de exemplo.
- [ ] Estratégia de importação genérica para usuários que já possuem dados em sistemas legados.

## Recommended next step

`/agile-epic` — A criação do Tenancit é uma iniciativa coordenada com múltiplas
dependências: definição de stack, modelo de dados, API REST, criptografia,
painel administrativo, autenticação, documentação e validação E2E.

## Verification

- [x] The problem is clear enough for the next step
- [x] Constraints and assumptions have been made explicit
- [x] The next artifact in the flow has been defined

<!-- Save to: planning/tenancit/intake.md -->
