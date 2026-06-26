# Story 5 — consumer-api-auth

**Origin:** `planning/tenancit/epics/01-tenancit/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** API de consumo (resolve por hostname) com decrypt server-side + auth por API key.
- **Refs:** RN-02, RN-05, RN-09; decisão #5/#8/#9.

## Traceability
- Business rules: RN-02 (resolução por hostname), RN-05 (decrypt server-side), RN-09 (API key hash).

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `server/internal/service/resolve.go` | Create | Resolver por hostname | core |
| `server/internal/service/apikey.go` | Create | hash + geração de token | core |
| `server/internal/httpapi/auth.go` | Create | middleware RequireAPIKey | core |
| `server/internal/httpapi/consumer.go` | Create | handlers resolve | core |
| `server/internal/httpapi/auth_test.go` | Create | testes de auth | core |
| `server/internal/httpapi/integration_test.go` | Create | E2E admin→resolve | core |

## Detail
### TO-BE
- `GET /v1/resolve?hostname=` → tenant + recursos ativos (secret em claro, RN-05).
- `GET /v1/resolve/{host}/resources/{definitionKey}` → recurso específico.
- Middleware API key (hash em api_clients): sem/!=token → 401; revogado → 401.

## Acceptance criteria
- [x] Resolve por match exato de hostname (RN-02).
- [x] Secrets retornam descriptografados (RN-05).
- [x] 401 sem token; 401 revogado; 200 com token ativo.

## Test-first plan
- **Behavior:** resolver decifra secret; middleware barra sem/revogado.
- **First failing test:** `auth_test.go` (401/200/revoked) + `integration_test.go` (E2E).
- **Level:** unit (auth) + integração (resolve com Postgres).

## Tasks
- [x] **Red:** auth_test + integration_test.
- [x] **Green:** resolve.go + apikey.go + auth.go + consumer.go.
- [x] **Refactor:** vet limpo.

## Verification
- [x] `go test ./internal/httpapi/` (com `TEST_DATABASE_URL`) — verde.
- [x] E2E container: resolve sem auth=401, com auth=200 + password decifrado.
