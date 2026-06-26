# Usability — Recuperar acesso após token inválido (admin-invalid-token-recovery)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado**; regressão do caminho 401 (ApiError) confirmada OK
- **Ambiente:** stack TenancitIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1. Sem token → tela **"Acesso administrativo"** com campo Token + Entrar. ✅
2–3. Token `rt_admin_invalido` + Entrar → app tenta autenticar, recebe **401**, **volta ao login** (shell não renderiza), exibe **"Token inválido ou expirado…"**, e **não salva** o token. ✅
4. Token `tenancit_admin_dev` + Entrar → Visão geral carrega com KPIs reais e sidebar. ✅
5. **Sair** → acesso removido, tela de login volta a bloquear. ✅

## Findings
Nenhum. **Regressão relevante verificada:** o caminho 401 agora passa pelo `ApiError(401)` + `notifyAdminAuthRequired("auth.invalidToken")` (mudança desta sessão) e continua exibindo a mensagem localizada de token inválido corretamente — sem ficar em estado parcialmente autenticado.
