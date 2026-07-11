# Usability — Resolver recurso específico por hostname e definition key (consumer-specific-resource-resolution)
- **Persona:** Integradora de serviço consumidor · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ⚠️ completable with friction

## Walkthrough
1. A autenticação admin abriu a visão geral e a navegação lateral.
2. Pela UI de **Recursos**, foram criadas as definitions `postgres-specific-290919` e `vault-specific-290919`, cada uma com um campo obrigatório.
3. Pela UI de **Tenants**, foi criado o tenant `specific-290919`.
4. Na aba **Domínios**, foi adicionado `specific-290919.localhost`.
5. Na aba **Recursos**, foram provisionados os recursos das duas definitions e ambos apareceram ativos no tenant.
6. Pela UI de **Chaves de API**, foi criado o client `consumer-specific-290919`; o token completo apareceu uma única vez no diálogo.
7. Em cliente HTTP externo, `GET /v1/resolve/specific-290919.localhost/resources/postgres-specific-290919` com `Authorization: Bearer <token>` retornou `200` com apenas o recurso `postgres-specific-290919`.
8. O mesmo endpoint com definition key inexistente retornou `404` com `resource not found`.
9. Após revogar a chave pela UI, a chamada válida passou a retornar `401` com `invalid api key`.

## Findings (prioritized)
| # | Severity | Step | What happened | Suggested fix |
|---|---|---|---|---|
| 1 | low | 6-7 | A tela **Chaves de API** exibe snippet apenas para `/v1/resolve?hostname=...`, mas o contrato também oferece `/v1/resolve/{hostname}/resources/{definitionKey}`. | Incluir o atalho por resource key no card **Exemplo de consumo** ou em uma opção copiável secundária. |

## Key screens

## Execution notes
- Browser plugin usado para a preparação administrativa do fluxo.
- Cliente HTTP externo usado para validar `/v1/resolve/{hostname}/resources/{definitionKey}`.
- Resultado funcional: `200` para resource específico, `404` para resource inexistente e `401` após revogação da API key.
