# Usability — Criar e manter resource definition (resource-definition-management)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — i18n/layout validation
1. A navegação lateral abriu **Recursos** e a lista carregou as definitions.
2. O botão **Nova definição** abriu o formulário com Key, Nome e Descrição.
3. A definition `Redis e2e-551174` foi criada e o app navegou para o detalhe.
4. **Novo campo** adicionou `host` com label `Host` e marcação **Obrigatório**.
5. **Novo campo** adicionou `password` com label `Password`, **Obrigatório** e **Segredo**.
6. **Desativar** mudou o status para `inativo`; **Ativar** retornou para `ativo`.
7. A lixeira removeu o campo `password`.
8. O retorno pela sidebar exibiu o card com `Campos: 1 · Segredos: 0`; os footers ficaram com altura consistente por linha.

## Post-fix validation
1. A ação **Remover** em campo abriu o diálogo **Remover campo?** e a ação foi cancelada sem alterar dados.
2. **Desativar** exibiu feedback **Definição desativada.**.
3. **Ativar** exibiu feedback **Definição ativada.** e restaurou a definição para ativa.

## Findings (prioritized)
Nenhum achado aberto após a validação pós-fix.

## Resolved findings
| # | Severity | Step | Previous finding | Resolution |
|---|---|---|---|---|
| 1 | medium | 7 | A remoção de campo era imediata, sem confirmação. | Adicionado diálogo de confirmação antes de remover campo. |
| 2 | low | 6-7 | Ativar/desativar e remover campo não exibiam confirmação textual. | Adicionado feedback textual para ações administrativas. |

## Key screens
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-definitions-list.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-new-definition-dialog-filled.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-definition-created.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-host-field-added.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-password-field-added.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-definition-inactive.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-definition-active.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-password-field-removed.png`
- `e2e/usability/screenshots/resource-definition-management-rerun-2026-06-24-definition-card-updated.png`

## Execution notes
- Browser plugin usado no fluxo `resource-definition-management`.
- Resultado funcional: fluxo completou sem erro bloqueante.
- Rerun validou os textos em pt-BR (`Nova definição`, `Segredos`) e o alinhamento/altura dos rodapés dos cards.
