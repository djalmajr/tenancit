# História 06 — Rate limit global com Valkey

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Aplicar RPM global por identidade em múltiplas réplicas.

## Rastreabilidade

Política de API clients, seção Rate limit.

## Arquivos

Interface/adapter do limiter, middleware, Compose/E2E e runbook.

## Detalhe

Token bucket atômico em Valkey; 429 com headers; indisponibilidade retorna 503
sem fallback local em produção.

## Tarefas

- [x] RED do algoritmo, headers e falha fechada.
- [x] Adapter Valkey e configuração segura.
- [x] Compose local/E2E e readiness.
- [x] Prova em duas instâncias/restart.

## Verificação

Unit, integração Valkey, multi-instância e teste de ausência de token nas keys.
