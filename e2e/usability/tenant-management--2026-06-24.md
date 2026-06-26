# Usability — Cadastrar e manter tenant com domínio (tenant-management)
- **Persona:** Operador técnico da plataforma · **Date:** 2026-06-24 · **Entry:** `http://localhost:5180/`
- **Verdict:** ✅ completable

## Rerun — tenant lifecycle validation
1. A navegação lateral abriu **Tenants** sem exigir URL manual.
2. A lista mostrou busca, tabela e botão **Novo tenant**.
3. O diálogo **Novo tenant** aceitou Nome e Slug únicos: `Tenant rerun-694226` e `tenant-rerun-694226`.
4. Após **Criar tenant**, a aplicação navegou para o detalhe do tenant recém-criado.
5. A ação **Editar** permitiu alterar o nome para `Tenant rerun-694226 Editado`, e o cabeçalho refletiu a alteração persistida.
6. Na aba **Domínios**, a ação **Adicionar** cadastrou `app-rerun-694226.local`, e o hostname apareceu na tabela.
7. Voltando para **Tenants**, a busca por `tenant-rerun-694226` filtrou a tabela para o tenant esperado.
8. Reabrindo o detalhe, a lixeira do domínio removeu o hostname, e o estado vazio voltou a informar que um domínio é necessário para resolver o tenant.

## Post-fix validation
1. A seção **Prontidão para consumo** apareceu no detalhe do tenant.
2. Um domínio temporário `cleanup-1782339046327.localhost` foi criado pela UI e exibiu **Domínio adicionado.**.
3. A ação **Remover** abriu o diálogo **Remover domínio?**.
4. A confirmação removeu o domínio temporário, exibiu **Domínio removido.** e a API confirmou que não restaram domínios no tenant de teste.

## Findings (prioritized)
Nenhum achado aberto após a validação pós-fix.

## Resolved findings
| # | Severity | Step | Previous finding | Resolution |
|---|---|---|---|---|
| 1 | medium | 8 | A remoção do domínio acontecia imediatamente, sem confirmação ou undo. | Adicionado diálogo de confirmação contextual antes de remover domínio. |
| 2 | low | 4-8 | As operações dependiam apenas da mudança visual de estado. | Adicionado feedback textual para edição, adição e remoção. |

## Key screens
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-tenants-list.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-new-tenant-dialog.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-tenant-detail-created.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-tenant-edited.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-domain-added.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-tenant-search.png`
- `e2e/usability/screenshots/tenant-management-rerun-2026-06-24-domain-removed.png`

## Execution notes
- Browser plugin usado no fluxo `tenant-management`.
- Dados criados: tenant `tenant-rerun-694226`; domínio `app-rerun-694226.local` foi removido ao final do fluxo.
- Resultado funcional: fluxo completou sem erro bloqueante.
