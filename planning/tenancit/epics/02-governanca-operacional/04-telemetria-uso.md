# História 04 — Telemetria de uso

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Dar visibilidade operacional sem colocar métricas no caminho de segurança.

## Rastreabilidade

Política de API clients, seção de uso e observabilidade.

## Arquivos

Store/worker de agregação, endpoints admin e dashboard frontend.

## Detalhe

Coalescer `last_used_at`, agregar uso diário por operação/classe e reter seis
meses. Polling ocorre somente com a aba visível.

## Tarefas

- [x] RED de concorrência, UTC, flush e retenção.
- [x] Agregador e job de limpeza.
- [x] APIs de usage/overview.
- [x] Dashboard e filtros.

## Verificação

Testes de corrida, integração, frontend e observabilidade do job.
