# Usability — Gerar, copiar, revogar e reativar API client (api-client-token-lifecycle)
- **Persona:** Integradora de serviço consumidor · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — nav/i18n validation
1. Sem token salvo, a rota `/api-clients` exibiu **Acesso administrativo**.
2. Após login, a navegação lateral apresentou **Chaves de API** e a tela manteve o título **Chaves de API**.
3. A tela exibiu aviso sobre acesso a segredos claros via API e listou clients existentes com token mascarado.
4. **Nova chave** abriu o diálogo de criação.
5. A chave `client-rerun-965549` gerou um token completo uma única vez.
6. **Copiar** mudou para **Copiado**, e o clipboard do Browser continha exatamente o token gerado.
7. **Concluir** fechou o diálogo; a tabela exibiu apenas `rt_live_••••••••`.
8. A ação de status revogou a chave para `revogado` e depois reativou para `ativo`.
9. O token completo não voltou a aparecer na listagem.

## Post-fix validation
1. A tela **Chaves de API** exibiu o card **Exemplo de consumo** com `Authorization: Bearer <token>` e `/v1/resolve?hostname=<tenant-hostname>`.
2. O botão **Copiar snippet** exibiu feedback **Snippet copiado**.

## Findings (prioritized)
Nenhum achado aberto após a validação pós-fix.

## Resolved findings
| # | Severity | Step | Previous finding | Resolution |
|---|---|---|---|---|
| 1 | low | 5-7 | Não havia exemplo imediato de uso do header `Authorization: Bearer <token>`. | Adicionado snippet de consumo na tela de chaves. |

## Key screens
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-auth-screen.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-api-clients-list.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-new-client-dialog.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-token-generated.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-token-copied.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-client-listed.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-client-revoked.png`
- `e2e/usability/screenshots/api-client-token-lifecycle-rerun-2026-06-24-client-reactivated.png`

## Execution notes
- Browser plugin usado no fluxo `api-client-token-lifecycle`.
- Resultado funcional: fluxo completou sem erro bloqueante; token copiado bateu com o token exibido e a listagem manteve apenas o preview mascarado.
