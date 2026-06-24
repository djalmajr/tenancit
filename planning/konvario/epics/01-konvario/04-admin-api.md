# Story 4 — admin-api

**Origin:** `planning/konvario/epics/01-konvario/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** API REST admin com CRUD + validação contra definition + mascaramento de secrets.
- **Refs:** RN-01, RN-03, RN-04, RN-06, RN-08, RN-09; proto `/tenants*`, `/resource-definitions*`, `/api-clients`.

## Traceability
- Business rules: RN-01, RN-03 (validação required), RN-04 (cripto), RN-06 (mascaramento), RN-08, RN-09.
- Prototype: rotas admin.

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `server/internal/httpapi/server.go` | Create | router admin/consumer + DI | core |
| `server/internal/httpapi/admin.go` | Create | handlers CRUD | core |
| `server/internal/service/values.go` | Create | encode/decode + mask/reveal | core |
| `server/internal/service/values_test.go` | Create | testes de valores | core |

## Detail
### TO-BE
- CRUD: tenants, domains, resource-definitions (+fields), tenant-resources (+values), api-clients.
- Create de tenant-resource: valida required (RN-03), cifra `is_secret` (RN-04), unicidade via banco (RN-01 → 409), definition inativa bloqueia (RN-08).
- Secrets mascarados por padrão (RN-06).

## Acceptance criteria
- [x] CRUD completo nas entidades.
- [x] Required ausente → 400; secret cifrado ao salvar.
- [x] 2º recurso ativo do mesmo tipo → 409.
- [x] Token de API gerado e exibido uma vez.

## Test-first plan
- **Behavior:** create de resource valida e cifra; unicidade barra duplicado.
- **First failing test:** `values_test.go` (encode/decode/mask) e E2E de integração.
- **Level:** unit (service) + integração (handlers, em `httpapi/integration_test.go`).

## Tasks
- [x] **Red:** values_test (plain vs secret, mask/reveal).
- [x] **Green:** service/values + handlers admin + router.
- [x] Integração admin→resolve coberta na Story 5.

## Verification
- [x] `go test ./internal/service/` e `./internal/httpapi/` — verde.
- [x] E2E no container: CRUD 201, RN-01 → 409.
