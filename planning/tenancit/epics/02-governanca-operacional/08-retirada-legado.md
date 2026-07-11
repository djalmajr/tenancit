# História 08 — Retirada do legado

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Encerrar a compatibilidade somente após inventário e rotação comprovados.

## Rastreabilidade

Política de API clients, fase contract.

## Arquivos

Migration contract, handlers/UI legados, ADRs, runbooks e handoff.

## Detalhe

Zerar `legacy_unbounded`, tornar política obrigatória, remover reativação e
validar backup/restore e rollback operacional antes do contract.

## Tarefas

- [ ] Relatório/gate de inventário zerado.
- [ ] Migration contract e remoção de compatibilidade.
- [ ] Ensaio de backup/restore e multi-réplica.
- [ ] Documentação e fechamento do epic.

## Verificação

Gates completos, migration contract, E2E, Browser e evidência operacional.
