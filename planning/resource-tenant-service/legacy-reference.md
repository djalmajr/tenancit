# Referência — análise do legado (front-manager-api + hyper-resource-tenant-lib)

Engenharia reversa feita durante o grilling (2026-05-30). Serve de **inspiração de
padrão**; o novo serviço **não** precisa ser compatível (ver [design.md](./design.md)).

## Repositórios

- `run2biz/front-manager-api` — API REST + service + entidades JPA (Spring Boot/Java).
- `run2biz/hyper-resource-tenant-lib` — lib de distribuição (Kafka) + manutenção de
  cache (MinIO) + cripto (`SecurityGenerator`).

## API atual (`/internal/resource-tenant`)

`ResourceTenantResource.java`:
- `GET /platform-application/{identificationKey}` → ativos por aplicação.
- `GET /platform-application/{identificationKey}/domain/{domain}` → ativos por domínio.
- `GET /platform-application/{identificationKey}/tenant/{tenant}` (+ `?resourceType`).
- `POST /platform-application/{identificationKey}/tenant/{tenant}` → create.
- `PUT .../tenant/{tenant}/resource-tenant-id/{id}/disable` → disable.

`identificationKey` (Base64) resolve `PlatformApplication`; `tenant` resolve via
`ClusterSpaceClient`.

## Fluxo de create (`ResourceTenantServiceImpl.create`)

1. `validateForCreation`.
2. Resolve `PlatformApplication` (`getByIdentificationKey`).
3. Resolve `ClusterSpaceClient` (`getByClusterSpaceUuid(tenant)`) → fornece
   `alias`, `mainUrlDns` (domain), `clusterSpaceUuid`.
4. Resolve `ResourceTemplate`.
5. Mapeia entidade, `enabled()`.
6. `encryptSensitiveDataField` — campos com `sensitiveData=true` passam por
   `securityGenerator.encrypt`.
7. `verifyContainsSomeEnabled` — impede 2º recurso ativo p/ mesmo
   tenant+template+app.
8. `save`.
9. Monta `ResourceTenantBody` e `resourceTenantKafkaService.createBlockSend(body)`
   (envio **bloqueante** com timeout).

`disable`: busca por clusterSpaceClient+platformApplication+id, filtra ativo,
`disable()`, save, `disableBlockSend`.

> **Dependências cross-domain importantes:** o fluxo depende de `ClusterSpaceClient` e
> `PlatformApplication`, que vivem no `front-manager-api`. Foi o que motivou o pivot
> para um serviço que é **dono** do conceito de Tenant.

## Contrato Kafka (lib `maintainer`)

- Mensagem: `ResourceTenantMessage` (não o `ResourceTenantBody` cru).
  - `Operation { CREATE, DISABLE }`, payload `current = ResourceTenantBody`.
  - Headers: `X-application-identificationKey`, `Cluster-Space` (tenant uuid).
  - Tópico: configurável (`applicationCachesTopic`).
- `ResourceTenantBody`: `id, templateName, templateVersion, alias, domain, tenant,
  active, appIdentificationKey, appName, fields[]`.
  - `Field`: `value, valueId, sensitive, settingId, settingKey, settingDescription`.

## Contrato de cache (MinIO)

`FileCacheServiceImpl`:
- Formato do arquivo: **YAML** (`ObjectMapper(new YAMLFactory())`), serializa
  `CacheFileObject { domain, alias, templateName, templateVersion, tenant, fields[] }`,
  `CacheField { key, sensitive, value }`.
- Path: `StorageFileManager.buildPathToFileCache(appName, tenant, templateName, templateVersion)`.
- Object tags: `resource-template`, `application-name`, `domain`, `tenant`.
- Create grava arquivo; Disable remove arquivo.

## Criptografia legada (NÃO portar)

`ObfuscateUtil` / `SecurityGenerator`:
- **DES** (chave efetiva 56 bits), `PBEWithMD5AndDES`, **20 iterações**, MD5.
- Senha embutida no fonte (`PASSWORD = "jtr0>eOwb!)8)k!E"`), versionada e replicada.
- Sem AEAD, sem rotação, prefixo fixo `!@,`.
- **Decisão:** substituído por AES-256-GCM com chave externalizada.

## Mapeamento legado → novo schema

| Legado | Novo (ver design.md) |
|--------|----------------------|
| `ClusterSpaceClient` | `tenants` (+ `tenant_domains`) |
| `ResourceTemplate` | `resource_definitions` |
| `ResourceFieldSetting` | `resource_fields` |
| `ResourceTenant` | `tenant_resources` |
| `ResourceTenantFieldValue` | `tenant_resource_values` |
| `message_key` | `resource_fields.key` |
| `sensitive_data` | `resource_fields.is_secret` |
| `PlatformApplication` | (removido — fora do escopo multi-app) |
