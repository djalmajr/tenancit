# Story 6 — spa-admin

**Origin:** `planning/konvario/epics/01-konvario/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** SPA admin real (React + shadcn + TanStack Router) consumindo a admin-api.
- **Refs:** proto (todas as rotas); RN-01/03/06.

## Traceability
- Prototype: Visão geral, Tenants, Tenant detail, Resource Definitions, API Clients.

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `web/package.json`, `tsconfig.json`, `vite.config.ts`, `index.html` | Create | bootstrap Vite+React+TS+Tailwind v4 | core |
| `web/src/lib/{utils,api}.ts` | Create | cn + client da admin-api | core |
| `web/src/components/ui/*.tsx` | Create | shadcn (button, card, input, badge, table) | core |
| `web/src/routes/*.tsx`, `router.tsx`, `main.tsx` | Create | rotas TanStack Router code-based | core |

## Detail
### TO-BE
- Rotas: `/`, `/tenants`, `/tenants/$id`, `/resource-definitions`, `/api-clients`.
- Telas com criação (tenant, definition, domínio) e listagens; client same-origin `/v1/admin`.

## Acceptance criteria
- [x] Rotas TanStack Router equivalentes às do protótipo.
- [x] Build (`bun run build`) gera `dist/`; `tsc --noEmit` limpo.
- [x] Consome a admin-api (same-origin), integrado via embed (Story 7).

## Test-first plan
- **Behavior:** SPA builda type-safe e renderiza as rotas.
- **Verificação primária (FE):** `tsc --noEmit` + build são o gate (casca/telas; sem lógica de negócio no front que justifique testes unitários de UI).
- **Low-value a evitar:** snapshots de UI; testar o router/framework.

## Tasks
- [x] Bootstrap web (Vite/React/TS/Tailwind v4/shadcn manual).
- [x] Componentes ui + client api.
- [x] Rotas + router + main.
- [x] Build verde.

## Verification
- [x] `bun run build` + `bunx tsc --noEmit` — verde (bundle ~313KB).
- [x] Servido pelo binário (Story 7), navegação client-side OK.
