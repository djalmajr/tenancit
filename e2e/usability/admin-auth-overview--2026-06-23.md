# Usability — Acessar painel administrativo e revisar visão geral (admin-auth-overview)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-24 rerun · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — i18n/theme/logout validation
1. **Sair** removeu o acesso administrativo e voltou para a tela dedicada **Acesso administrativo**, sem renderizar o shell por trás.
2. O menu de idioma no login alternou **Português**, **English** e **Español**; o trigger fechado exibiu apenas a bandeira.
3. O menu de tema no login alternou **Claro**, **Escuro** e **Sistema** sem deslocar o card.
4. O token `konvario_admin_dev` carregou a visão geral com KPIs reais.
5. O header autenticado alternou idioma e tema; navegação e conteúdo acompanharam o idioma.
6. A sidebar navegou por **Visão geral**, **Tenants**, **Recursos** e **Chaves de API**.
7. A sidebar colapsada manteve `aria-label` nos links; **Sair** no rodapé voltou ao login e removeu o shell.

## Findings (prioritized)
| # | Severity | Step | What happened | Suggested fix |
|---|---|---|---|---|
| 1 | resolved | 1 | A tela sem token bloqueia o painel e não mostra KPIs por trás. | Mantido. |
| 2 | resolved | 6 | Os links colapsados mantêm nomes acessíveis por `aria-label`. | Mantido. |
| 3 | resolved | 7 | O painel tem ação explícita **Sair**. | Mantido. |

## Key screens
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-pt.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-language-menu.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-en.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-es.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-theme-menu.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-login-dark.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-overview-after-login.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-overview-en.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-overview-es.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-collapsed-sidebar.png`
- `e2e/usability/screenshots/admin-auth-overview-rerun-2026-06-24-after-logout.png`

## Execution notes
- Browser plugin usado no fluxo `admin-auth-overview`.
- KPIs observados no rerun: `Tenants ativos=5`, `Domínios=6`, `Recursos provisionados=5`, `Definições ativas=8`.
- Resultado funcional: fluxo completou sem erro bloqueante.
