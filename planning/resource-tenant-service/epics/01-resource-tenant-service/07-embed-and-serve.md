# Story 7 — embed-and-serve

**Origin:** `planning/resource-tenant-service/epics/01-resource-tenant-service/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** Servir o SPA pelo binário Go (embed.FS), mesmo origin da API, fallback p/ index.html.
- **Refs:** decisão #13.

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `server/internal/spa/spa.go` | Create | `//go:embed all:dist` + fallback | core |
| `server/internal/spa/dist/*` | Generate | SPA buildado (via `make embed`) | probable |
| `server/cmd/server/main.go` | Modify | monta API + static handler | core |
| `Makefile` | Modify | alvo `embed` (web/dist → spa/dist) | core |

## Detail
### TO-BE
- `/` serve SPA; `/v1/*` serve API; assets reais servidos direto; demais paths → index.html.
- Sem CORS (mesmo origin).

## Acceptance criteria
- [x] `go build` embute `web/dist`; binário serve SPA + API.
- [x] Fallback SPA (ex.: `/tenants` → index.html, 200).
- [x] Binário sobe, serve index.html e responde `/v1/resolve` (401 sem key).

## Test-first plan
- **Behavior:** binário único serve estático + API no mesmo origin com fallback.
- **Verificação:** smoke do binário (health, `/`, `/tenants`, `/v1/resolve`).

## Tasks
- [x] spa.go com embed + fallback.
- [x] main.go integra Routes(staticHandler).
- [x] Makefile `embed`.

## Verification
- [x] Binário local: health=200, SPA com assets, `/tenants`=200, resolve sem auth=401.
