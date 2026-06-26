# Usability — Provisionar recurso do tenant e validar segredos (tenant-resource-lifecycle)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — resource/i18n validation
1. A precondição criou tenant `res-rerun-047968` e definition `vault-rerun-047968` com campos `host` e `password` secreto.
2. Pela UI, a navegação lateral abriu **Tenants**, e a busca encontrou `res-rerun-047968`.
3. O detalhe abriu na aba **Recursos** com estado vazio.
4. **Adicionar recurso** mostrou a definition ativa disponível.
5. Ao selecionar `Vault rerun-047968`, o formulário dinâmico exibiu `Host` e `Password`.
6. Após preencher `vault-rerun-047968.internal` e `<synthetic-secret>`, **Salvar recurso** criou o recurso.
7. O valor secreto permaneceu mascarado na listagem inicial.
8. **Revelar segredos** carregou os valores claros no backend, mas a UI ainda exigiu clique explícito em **Revelar** para mostrar o segredo.
9. **Desativar** mudou o recurso para `inativo`; **Reativar** voltou para `ativo`.
10. **Remover** apagou o recurso e o estado vazio voltou a aparecer.
11. Pós-patch, o badge do campo secreto passou a exibir **Segredo** em vez de `secret`.

## Post-fix validation
1. O detalhe de tenant exibiu **Prontidão para consumo** com contagem de recursos ativos.
2. O botão global passou a exibir **Habilitar revelação de segredos**.
3. A ação **Remover** em recurso abriu o diálogo **Remover recurso?** e foi cancelada sem alterar dados.
4. **Desativar** exibiu **Recurso desativado.**; **Reativar** exibiu **Recurso reativado.** e restaurou o recurso para ativo.

## Findings (prioritized)
Nenhum achado aberto após a validação pós-fix.

## Resolved findings
| # | Severity | Step | Previous finding | Resolution |
|---|---|---|---|---|
| 1 | medium | 10 | Remover recurso era uma ação destrutiva imediata, sem confirmação. | Adicionado diálogo de confirmação contextual antes de remover recurso. |
| 2 | low | 8 | O botão global **Revelar segredos** não explicava a segunda etapa por campo. | Copy ajustada para **Habilitar revelação de segredos** e feedback de habilitação. |

## Key screens
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-tenant-search.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-tenant-detail-empty-resources.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-dialog-fields.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-created-masked.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-secrets-loaded-still-hidden.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-secret-revealed.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-inactive.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-active.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-removed.png`
- `e2e/usability/screenshots/tenant-resource-lifecycle-rerun-2026-06-24-resource-created-secret-badge-i18n.png`

## Execution notes
- Browser plugin usado no fluxo `tenant-resource-lifecycle`.
- Precondição criada pela Admin API com `konvario_admin_dev`.
- Resultado funcional: fluxo completou sem erro bloqueante; segredo ficou oculto por padrão e só apareceu após ação explícita por campo.
