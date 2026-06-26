# Usability — Navegação e operação em viewport mobile (responsive-mobile-navigation)
- **Persona:** mobile · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/ (375×812)
- **Verdict:** ⚠️→✅ — 1 achado (sheet não fechava ao navegar), **corrigido e verificado**
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433), viewport mobile 375px.

## Walkthrough
1. **auth** — card "Acesso administrativo" cabe em 375px; idioma/tema acionáveis. ✅
2. **shell** — sidebar fica offcanvas (nav não inline); "Toggle sidebar" abre a navegação como **sheet** sobre o conteúdo (drawer à esquerda + overlay). ✅
3. **shell** — tocar **Tenants** navega corretamente, MAS **o sheet permanecia aberto** cobrindo o conteúdo (rota mudava por baixo). ❌→✅ **corrigido**.
4. **tenants-list** — tabela com `overflow-x: auto` (rola horizontalmente sem quebrar a página; `main` não excede a viewport); **Novo tenant** acionável; paginação reflui empilhada. ✅
5. **tenant-detail** — header, breadcrumb e card **Prontidão para consumo** (grid 4 col) **empilham em 1 coluna**; diálogo **Adicionar recurso** cabe na viewport (375px) e é operável. ✅
6. **definitions-list** — grade de cards reflui para **1 coluna** (cards ~327px). ✅
7. **api-clients** — criar chave + diálogo de token: **cabe em 375px**, token completo (62 chars) **não cortado**, botão Copiar presente. ✅

## Findings (prioritized)
| # | Severity | Step | What happened | Fix |
|---|---|---|---|---|
| 1 | medium | 3 | Em mobile, ao tocar um item no sheet de navegação, a rota mudava mas **o sheet não fechava** — ficava sobre o conteúdo, exigindo um toque extra (overlay/Esc) para ver a tela escolhida. | **CORRIGIDO** — `SidebarMenuButton` (`ui/sidebar.tsx`) agora fecha o sheet mobile (`setOpenMobile(false)`) ao ser ativado, preservando o `onClick` original; cobre nav e logout. Re-teste: tocar Tenants → sheet fecha e o conteúdo aparece. |

## Observações
- A sobreposição visual nome/slug numa linha da tabela de tenants vem do tenant de nome artificialmente gigante criado no seed do flow 3 — não é bug de layout (a tabela rola horizontalmente); um nome normal não sobrepõe.
- Reflow de grids (`sm/lg/md:grid-cols-*` → 1 coluna) e fit de diálogos em 375px estão corretos.
