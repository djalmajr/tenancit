# Usability — Recuperar acesso após token administrativo inválido (admin-invalid-token-recovery)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ⚠️ completable with friction

## Walkthrough
1. Sem token salvo, a tela dedicada **Acesso administrativo** apareceu com campo **Token**, alternadores de preferência e botão **Entrar**.
2. Ao informar `rt_admin_invalido` e clicar em **Entrar**, a aplicação retornou para a tela de acesso, sem vazar shell ou dashboard por trás.
3. A mensagem exibida foi **"Informe o token administrativo para acessar o painel."**; o operador entende que precisa corrigir a credencial, mas a causa não fica explícita.
4. Ao informar `tenancit_admin_dev` e clicar em **Entrar**, a visão geral carregou com KPIs reais e navegação lateral.
5. Ao clicar em **Sair**, a tela de **Acesso administrativo** voltou a bloquear o painel.

## Findings (prioritized)
| # | Severity | Step | What happened | Suggested fix |
|---|---|---|---|---|
| 1 | low | 3 | Após token inválido, a mensagem fala em token necessário, mas não confirma que o token informado foi recusado. | Trocar a cópia de recuperação para algo como **"Token inválido ou expirado. Informe um token administrativo válido."** |

## Key screens
- `e2e/usability/screenshots/admin-invalid-token-recovery-2026-06-24-auth.png`
- `e2e/usability/screenshots/admin-invalid-token-recovery-2026-06-24-invalid-token.png`
- `e2e/usability/screenshots/admin-invalid-token-recovery-2026-06-24-overview.png`
- `e2e/usability/screenshots/admin-invalid-token-recovery-2026-06-24-after-logout.png`

## Execution notes
- Browser plugin usado no fluxo `admin-invalid-token-recovery`.
- Resultado funcional: token inválido não libera o painel; token válido libera; logout remove o acesso.
