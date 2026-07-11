# História 11 — Gate de escala e contract final

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Encerrar o epic com decisão baseada em volume observado, retirada de caminhos
transitórios e documentação da verdade executável.

## Responsabilidade, motivação e valor

Esta história é um freio contra complexidade prematura. Mede cardinalidade,
payload, latência e memória antes de decidir por paginação server-side ou novos
contratos, e remove compatibilidades que só existiam durante migração.

**Ganho:** o Tenancit escala quando a evidência justificar. `KEEP_FULL_LISTS` é
um resultado tão válido quanto `MIGRATE`; nenhuma arquitetura entra por
especulação.

## Arquivos

- Benchmarks/gate de escala, APIs/tabelas afetadas se `MIGRATE`.
- ADRs, contratos, runbooks, handoff, epic statuses e evidências de release.

## Detalhe

Medir cardinalidade, payload, p95, memória do browser e custo de consulta. Só
implementar paginação server-side quando limiares aprovados forem excedidos.
Auditoria e change feed já são cursor-based por contrato.

## Tarefas

- [ ] Coletar volume real/previsto e executar benchmark reproduzível.
- [ ] Registrar decisão `KEEP_FULL_LISTS` ou `MIGRATE` por superfície.
- [ ] Se necessário, implementar endpoints paginados sem quebrar filtros/deep links.
- [ ] Remover compatibilidade temporária e flags cujo inventário chegou a zero.
- [ ] Reexecutar todos os gates, restore, rollback e E2E stability.
- [ ] Atualizar documentos e marcar histórias pelo estado real.

## Verificação

Relatório de escala, três runs CI/E2E sem retry, deploy/restore/rollback
evidence e auditoria de consistência documental.
