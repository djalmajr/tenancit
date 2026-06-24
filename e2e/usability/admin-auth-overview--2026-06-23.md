# Usability — Acessar painel administrativo e revisar visão geral (admin-auth-overview)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-23 · **Entry:** `http://127.0.0.1:5180/`
- **Verdict:** ✅ completable

## Walkthrough
1. Sem token salvo, a entrada `/` abriu uma tela dedicada **"Acesso administrativo"** em card central, sem renderizar sidebar ou dashboard por trás.
2. Ao preencher **Token** com `rt_admin_dev` e clicar em **Entrar**, a visão geral carregou dados reais.
3. Os cards de KPI exibiram **Tenants ativos**, **Domínios**, **Recursos provisionados** e **Definitions ativas** com os valores `1`, `4`, `4` e `4`.
4. O card **"Tenants"** exibiu o tenant **CitPeople**, host primário `citpeople.localhost`, `4 recursos` e status `active`.
5. A sidebar expandida navegou corretamente por **Visão geral**, **Tenants**, **Resource Definitions** e **API Clients**; em todos os casos o título do header acompanhou a seção.
6. Ao colapsar a sidebar, os links continuaram clicáveis por ícone e mantiveram nomes acessíveis para **Visão geral**, **Tenants**, **Resource Definitions** e **API Clients**.
7. O botão **Sair** apareceu no rodapé da sidebar após autenticação; ao clicar, a página recarregou, removeu o estado autenticado da UI e voltou para a tela dedicada **"Acesso administrativo"**.

## Findings (prioritized)
| # | Severity | Step | What happened | Suggested fix |
|---|---|---|---|---|
| 1 | P2 corrigido | 1 | Antes do ajuste, a tela sem token renderizava KPIs zerados e **"Nenhum tenant ainda"** atrás do modal, o que poderia parecer estado real se o operador cancelasse o login. | Corrigido em `web/src/components/app-shell.tsx`: sem token ou após 401, o shell renderiza somente a tela dedicada de acesso. |
| 2 | P2 corrigido | 6 | Antes do ajuste, a sidebar colapsada escondia o texto visual com `display: none` e os links ficavam sem nome acessível no snapshot. | Corrigido em `web/src/components/app-shell.tsx`: os links de navegação recebem `aria-label` estável com o rótulo do item. |
| 3 | P2 corrigido | 7 | O painel persistia o token administrativo sem oferecer uma ação explícita para encerrar a sessão. | Corrigido em `web/src/components/app-shell.tsx`: o rodapé da sidebar exibe **Sair** quando existe token salvo; o clique limpa o token e recarrega a página. |

## Key screens
- `e2e/usability/screenshots/admin-auth-overview-2026-06-23-auth-card-cpanel-style.png`
- `e2e/usability/screenshots/admin-auth-overview-2026-06-23-overview-after-fix.png`
- `e2e/usability/screenshots/admin-auth-overview-2026-06-23-logout-sidebar-cpanel-style.png`
- `e2e/usability/screenshots/admin-auth-overview-2026-06-23-collapsed-sidebar-final.png`
- `e2e/usability/screenshots/admin-auth-overview-2026-06-23-after-logout.png`

## Execution notes
- Browser plugin usado no fluxo `admin-auth-overview`.
- Frontend: `http://127.0.0.1:5180/` com Vite apontando para API dev em `http://localhost:8083`.
- Console do Browser: nenhum erro após a revalidação final.
