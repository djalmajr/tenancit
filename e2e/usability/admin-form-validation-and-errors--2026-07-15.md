# Usability — Validação de formulários e erros de unicidade no painel admin (admin-form-validation-and-errors)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `partial`

## Evidência

Em Configurações, retenção `0` foi rejeitada com a mensagem `Dados inválidos. Revise os campos.` e sem persistência.

## Limite ou achado

Os casos de unicidade de tenant, definition e chave exigiriam mutações no conjunto compartilhado.

## Próximo passo

Executar os conflitos sobre dados descartáveis preparados para E2E.
