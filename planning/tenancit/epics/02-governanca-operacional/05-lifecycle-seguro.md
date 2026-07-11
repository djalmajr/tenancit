# História 05 — Lifecycle seguro

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Completar rename, policy update, rotação, revogação terminal e delete seguro.

## Rastreabilidade

Política de API clients e auditoria administrativa.

## Arquivos

Handlers/store de API clients e console `/api-clients`.

## Detalhe

Token bruto apenas em create/rotate; delete somente revogado; expirado exige
rotação; todas as ações são auditadas.

## Tarefas

- [x] RED de idempotência, conflitos e token one-shot.
- [x] Endpoints PATCH/rotate/revoke/delete.
- [x] UI de ações e janela de transição.
- [x] Remover toggle após gate do legado.

## Verificação

Unit, integração, E2E lifecycle e canários de token.
