# Story 1 — bootstrap-monorepo

**Origin:** `planning/resource-tenant-service/epics/01-resource-tenant-service/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** Casca do monorepo (`server/` Go + `web/` SPA), build/test únicos, health com TDD.
- **Constraints:** Go 1.25 (directive), Bun, Docker. Stack fixada (decisões #6/#7/#13).
- **Refs:** wiki `architecture/overview`, `decisions/decision-log`.

## Traceability
- Prototype: alvo final do `web/` (todas as rotas) — nesta story só a casca.
- Source docs: `architecture/overview`, decisões #6/#7/#13.

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `go.work` | Create | workspace Go do monorepo | core |
| `server/go.mod` | Create | módulo Go do server | core |
| `server/cmd/server/main.go` | Create | entrypoint HTTP | core |
| `server/internal/httpapi/health.go` | Create | handler `GET /healthz` | core |
| `server/internal/httpapi/health_test.go` | Create | teste TDD do health | core |
| `server/internal/httpapi/server.go` | Create | montagem do chi router (`NewRouter`) | core |
| `web/*` (Vite+React+TS+TanStack Router+Tailwind v4) | Create | SPA | core |
| `Makefile` | Create | build/test únicos | core |

## Detail
### AS-IS
- Só `planning/` no repo; sem código.
### TO-BE
- `server/` compila e roda `GET /healthz` → 200 `{"status":"ok"}`.
- `web/` builda para `dist/`; `make build`/`make test` funcionam.
### Scope
- Includes: casca, health com teste, build único, SPA mínimo.
- Does not include: migrations, API real, telas, embed, docker (stories seguintes).

### Approach
- Go workspace + chi; health testado via `httptest`. Web Vite+React+TanStack Router+Tailwind v4 (shadcn manual).
> Nota de execução: o `NewRouter` ficou em `server.go` (o `router.go` inicial foi consolidado lá ao
> evoluir para o router completo na Story 4/5). Não há `router.go` separado.

## Acceptance criteria
- [x] `go build ./...` ok e `go test ./...` verde (health).
- [x] `bun run build` gera `web/dist/index.html`.
- [x] `make build` e `make test` rodam server+web.
- [x] `GET /healthz` → 200 `{"status":"ok"}` (provado por teste antes da implementação).

## Test-first plan
- **Behavior:** `GET /healthz` responde 200 e `{"status":"ok"}`.
- **First failing test:** `health_test.go` via `httptest` — falha sem o handler (Red).
- **Level:** unit + smoke.

## Tasks
- [x] `.gitignore` + `go.work`.
- [x] **Red:** `health_test.go` esperando 200 `{"status":"ok"}`.
- [x] **Green:** `health.go` + router (`server.go`/`NewRouter`) + `main.go`.
- [x] Scaffolding `web/` + build gera `dist/`.
- [x] `Makefile` com build/test/dev.
- [x] **Refactor:** `go vet`/`tsc --noEmit` limpos.

## Verification
- [x] `go vet ./... && go test ./...`
- [x] `bun run build && bunx tsc --noEmit`
- [x] `make build && make test`
- [x] Evidência Red→Green registrada
