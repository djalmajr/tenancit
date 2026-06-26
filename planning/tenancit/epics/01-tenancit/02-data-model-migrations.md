# Story 2 — data-model-migrations

**Origin:** `planning/tenancit/epics/01-tenancit/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** Schema PostgreSQL (goose, embutido) + acesso type-safe (sqlc) para todo o modelo.
- **Constraints:** snake_case, uuid PK, invariante RN-01 no banco.
- **Refs:** wiki `architecture/data-model`, `business/rules` (RN-01/07/08/09).

## Traceability
- Business rules: RN-01 (unicidade), RN-07 (soft-delete via status), RN-08, RN-09 (keys estáveis).
- Source docs: `architecture/data-model`.

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `server/migrations/00001_init.sql` | Create | DDL de todas as tabelas + índice único parcial | core |
| `server/migrations/embed.go` | Create | `//go:embed *.sql` | core |
| `server/internal/store/migrate.go` | Create | `Migrate(dsn)` via goose + pgx stdlib | core |
| `server/sqlc.yaml` | Create | config sqlc (pgx/v5, uuid, timestamptz) | core |
| `server/internal/store/queries/*.sql` | Create | queries tenants/definitions/resources | core |
| `server/internal/store/db/*` | Generate | código sqlc | probable |
| `server/internal/store/migrate_test.go` | Create | teste de schema + RN-01 | core |

## Detail
### AS-IS
- Sem persistência; só health/router.
### TO-BE
- Migrations criam `tenants, tenant_domains, resource_definitions, resource_fields, tenant_resources, tenant_resource_values, api_clients`.
- Índice único parcial `(tenant_id, resource_definition_id) WHERE status='active'` (RN-01).
- sqlc gera repositório type-safe.
### Scope
- Includes: DDL, migrate runner, queries, código gerado, teste DB.
- Does not include: handlers, cripto (outras stories).

## Acceptance criteria
- [x] Tabelas criadas pela migration.
- [x] RN-01: 2º recurso ativo viola índice; inativo permitido.
- [x] `tenant_domains.hostname` unique; `resource_definitions.key` unique; `resource_fields (definition_id,key)` unique.
- [x] sqlc compila e round-trip (CreateTenant→AddTenantDomain→GetTenantByHostname) verde.

## Test-first plan
- **Behavior:** migration cria schema e o banco rejeita 2º recurso ativo do mesmo tipo.
- **First failing test:** `migrate_test.go` espera as tabelas e a violação RN-01 — falha sem a migration.
- **Level:** integração (Postgres real via `TEST_DATABASE_URL`).

## Tasks
- [x] DDL + embed + runner.
- [x] **Red:** `migrate_test.go` (tabelas + RN-01). **Done when:** falha sem migration.
- [x] **Green:** migration aplica; teste passa.
- [x] sqlc.yaml + queries + geração; round-trip test.
- [x] **Refactor:** `go vet` limpo.

## Verification
- [x] `make test-db` (sobe Postgres efêmero) — verde.
- [x] RN-01 comprovada por teste.
