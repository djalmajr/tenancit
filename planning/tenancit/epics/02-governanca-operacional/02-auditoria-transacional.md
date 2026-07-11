# História 02 — Auditoria transacional

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Registrar mutações e reveal com principal verificável e sem material secreto.

## Rastreabilidade

`docs/developers/design/admin-audit-log.md`; ADR 0005.

## Arquivos

Backend HTTP/store, migration de partições e rota frontend de auditoria.

## Detalhe

Eventos append-only, metadata tipada, sucesso na mesma transação do domínio e
consulta keyset. O token compartilhado identifica a credencial, não uma pessoa.

## Tarefas

- [x] RED de rollback, reveal e redação.
- [ ] Writer transacional e cobertura dos dois cortes.
- [ ] API de consulta e tela filtrável.
- [ ] Runbook de partições/retenção.

## Verificação

Unit, DB, integração HTTP, Playwright e canários de secrets.
