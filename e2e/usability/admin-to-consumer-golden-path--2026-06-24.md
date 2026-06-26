# Usability — Configurar tenant no admin e resolver por hostname na Consumer API (admin-to-consumer-golden-path)
- **Persona:** Integradora de serviço consumidor · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — admin-to-consumer validation
1. Pela UI, a navegação abriu **Recursos**.
2. Foi criada a definition `golden-rerun-278554` com campos obrigatórios `host` e `password`, sendo `password` segredo.
3. Pela UI, foi criado o tenant `golden-rerun-278554`.
4. Na aba **Domínios**, foi adicionado `golden-rerun-278554.localhost`.
5. Na aba **Recursos**, foi provisionado o recurso `Golden rerun-278554` com `host=service-rerun-278554.internal` e segredo mascarado no painel.
6. Pela UI de **Chaves de API**, foi criada a chave `consumer-rerun-278554`, o token foi gerado e copiado.
7. Em cliente HTTP externo, `GET /v1/resolve?hostname=golden-rerun-278554.localhost` com `Authorization: Bearer <token>` retornou `200`, tenant slug correto e o recurso ativo com segredo descriptografado.
8. A mesma chamada sem token retornou `401`.
9. A chamada com hostname desconhecido e token válido retornou `404`.

## Post-fix validation
1. O detalhe de tenant passou a exibir **Prontidão para consumo** com tenant, domínios, recursos ativos e chaves ativas.
2. A tela **Chaves de API** passou a exibir **Exemplo de consumo** com snippet de `/v1/resolve`.
3. O clique em **Copiar snippet** exibiu feedback **Snippet copiado**.

## Findings (prioritized)
Nenhum achado aberto após a validação pós-fix.

## Resolved findings
| # | Severity | Step | Previous finding | Resolution |
|---|---|---|---|---|
| 1 | medium | 1-7 | Não havia superfície de checklist ou prontidão do tenant para consumo. | Adicionado card **Prontidão para consumo** no detalhe do tenant. |
| 2 | low | 7 | A Consumer API não tinha apoio visual no painel. | Adicionado snippet copiável de `/v1/resolve` em **Chaves de API**. |

## Key screens
- `e2e/usability/screenshots/admin-to-consumer-golden-path-rerun-2026-06-24-definition-ready.png`
- `e2e/usability/screenshots/admin-to-consumer-golden-path-rerun-2026-06-24-tenant-created.png`
- `e2e/usability/screenshots/admin-to-consumer-golden-path-rerun-2026-06-24-domain-added.png`
- `e2e/usability/screenshots/admin-to-consumer-golden-path-rerun-2026-06-24-resource-added.png`
- `e2e/usability/screenshots/admin-to-consumer-golden-path-rerun-2026-06-24-api-client-created.png`

## Execution notes
- Browser plugin usado para a parte administrativa do fluxo.
- Cliente HTTP externo usado para validar `/v1/resolve`.
- Resultado funcional: `200` com token válido, `401` sem token e `404` para hostname desconhecido.
