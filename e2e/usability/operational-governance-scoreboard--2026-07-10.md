# Scoreboard — governança operacional

- **Data:** 2026-07-10
- **Superfície:** Vite/proxy e API/Valkey em Compose
- **Verdict:** aprovado sem blockers; automação cobre os contratos críticos

| Persona | Fluxos | Blocker | High | Medium | Low | Evidência / resolução |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| Administrador (`platform-operator`) | criar, editar, rotacionar, revogar, excluir | 0 | 0 | 0 | 0 | lifecycle terminal automatizado; token one-shot e preview mascarado |
| Operador futuro (`future-operator`) | consultar uso, investigar auditoria, expiração | 0 | 0 | 0 | 1 | UI explicita que token compartilhado é credencial, não pessoa; identidade individual aguarda ADR 0005 |
| Integrador (`service-integrator`) | scope insuficiente, expiração, rate limit | 0 | 0 | 0 | 0 | matriz HTTP e teste Playwright validam `401/403/429`, headers e sinais de uso |

## Checkpoints

- Keyboard e mobile permanecem no catálogo existente.
- Textos novos existem em pt-BR, en-US e es-ES.
- Tabela permite filtros operacionais e persiste visibilidade de colunas no browser.
- Paginação permanece client-side: o checkpoint de volume não justificou mudar
  o contrato.
- O item low é uma dependência de identidade futura, não um defeito da entrega
  atual; nenhum evento atribui uma pessoa ao token compartilhado.
