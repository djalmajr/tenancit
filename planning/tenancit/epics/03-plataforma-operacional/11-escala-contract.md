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

- [x] Coletar volume real/previsto e executar benchmark reproduzível.
- [x] Registrar decisão `KEEP_FULL_LISTS` ou `MIGRATE` por superfície.
- [x] Se necessário, implementar endpoints paginados sem quebrar filtros/deep links.
- [x] Remover compatibilidade temporária e flags cujo inventário chegou a zero.
- [x] Reexecutar todos os gates, restore, rollback e E2E stability.
- [x] Atualizar documentos e marcar histórias pelo estado real.

## Verificação

Relatório de escala, três runs CI/E2E sem retry, deploy/restore/rollback
evidence e auditoria de consistência documental.

## Evidência entregue

- Não há cardinalidade operacional de produção fornecida; o gate registra esse
  fato como volume observado `0`, sem converter ausência de telemetria em prova
  de folga. O benchmark limpo em `301b470` mediu 100/500/1.000/5.000, duas
  rodadas por escala, e decidiu `KEEP_FULL_LISTS` nas quatro superfícies.
- O primeiro hard trigger repetível apareceu em 500 definições. O relatório
  `benchmarks/scale/report-2026-07-11.md` reduz esse checkpoint sem promover a
  curva sintética a necessidade atual; por isso nenhum endpoint paginado foi criado.
- O runner foi reconciliado com operations token, schema governado de API
  clients e seletores acessíveis. A API/SPA retiraram `legacy_unbounded`; preview,
  RPM e expiração são obrigatórios no tipo público e no contract SQL.
- `make test-db`, `make build`, rollback/deploy contracts e continuidade com
  duas réplicas passaram. O script de continuidade passou a enviar a chave
  idempotente obrigatória.
- Três stacks E2E novas passaram 22/22 + route smoke, sem retry. Backup e restore
  com cliente PostgreSQL 16 preservaram 22 tabelas e um tenant; o backup agora
  suporta `shasum` e `sha256sum` com checksum obrigatório.
