# Usability — Recuperar acesso após token administrativo inválido (admin-invalid-token-recovery)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `needs_ux_fix`

## Evidência

Após logout, o token inválido `tnc_admin_invalido` foi submetido e o login permaneceu bloqueado; em seguida o token válido recuperou o acesso.

## Limite ou achado

A tela limpou o campo, mas não exibiu mensagem visível informando que o token era inválido.

## Próximo passo

Exibir erro localizado junto ao campo e manter foco/descrição acessível após o `401`.
