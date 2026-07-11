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

- [ ] RED do algoritmo, headers e falha fechada.
- [ ] Adapter Valkey e configuração segura.
- [ ] Compose local/E2E e readiness.
- [ ] Prova em duas instâncias/restart.

## Verificação

Unit, integração Valkey, multi-instância e teste de ausência de token nas keys.
