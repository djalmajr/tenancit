# Story 8 — docker-image

**Origin:** `planning/tenancit/epics/01-tenancit/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** Dockerfile multi-stage + validação E2E com Postgres.
- **Refs:** `architecture/why-this-service` (leveza/deploy).

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `Dockerfile` | Create | multi-stage (bun build → go estático → distroless) | core |
| `docker-compose.yml` | Create | app + postgres healthcheck | core |
| `.dockerignore` | Create | exclui node_modules/dist/go.work/segredos | core |

## Detail
### TO-BE
- Estágio web (bun build) → estágio Go (build estático, embed) → runtime distroless.
- `docker compose up` sobe app+postgres; migrations aplicam; health ok.

## Acceptance criteria
- [x] Multi-stage; imagem mínima (distroless static).
- [x] compose sobe app + postgres; health 200.
- [x] E2E no container: criar tenant+recurso via admin e resolver por hostname.
- [x] Imagem documentada (tamanho/variáveis).

## Test-first plan
- **Behavior:** imagem sobe com Postgres e o fluxo admin→resolve funciona ponta-a-ponta.
- **Verificação:** script E2E (`/tmp/e2e.sh`) contra o container.

## Tasks
- [x] Dockerfile multi-stage + .dockerignore.
- [x] docker-compose com healthcheck.
- [x] Build + validação E2E.

## Verification
- [x] `make docker` constrói a imagem (**13.6MB**, distroless static).
- [x] `docker compose up`: health 200; E2E 13/14 checks PASS (1 "fail" é assert do script, não bug — 400/RN-03 antes de 409).
- [x] Cripto em repouso: password só em value_cipher (kv=1); 0 senhas em claro.

## Notas operacionais (gotchas)
- directive `go 1.25.x` no go.mod (base `golang:1.25-alpine` recusa `>=1.26.3`).
- `sqlc` fora do require do go.mod (quebra `go mod download` no build); `make sqlc` usa `go run ...@v1.30.0`.
- goose+sqlc sem `StatementBegin/End`.
