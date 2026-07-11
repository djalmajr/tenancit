# História 04 — Outbox, webhooks e change feed

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Consumidores precisam reagir a mudanças sem polling agressivo. A entrega deve
ser mais rigorosa que o dispatcher best-effort observado no reference implementation.

## Rastreabilidade

- Mutações de tenant, domínio, definition, field, resource e API client.
- Nova área `/integrations/webhooks`.

## Arquivos

- Migrations de outbox, targets, deliveries e DLQ.
- Publisher transacional, worker, assinatura, adapters e APIs/UI.
- Docs de eventos, segurança, operação e exemplos.

## Detalhe

Evento é escrito na transação de domínio. Worker usa lease/`SKIP LOCKED`, chave
idempotente, retry exponencial com jitter, timeout, circuit breaker e DLQ.
Payload tem versão, event ID, timestamp e referências; secrets nunca entram.
Formatos genérico, Slack, Discord e Teams são renderizadores opcionais.

## Tarefas

- [x] Versionar catálogo e schema dos eventos.
- [x] Implementar outbox em todas as mutações relevantes.
- [x] Implementar targets com URL cifrada, allowlist de schemes e SSRF defense.
- [x] Assinar timestamp/body e documentar replay window.
- [x] Implementar worker, retry, DLQ, replay autorizado e retenção.
- [x] Expor change feed cursor-based para consumidores autorizados.
- [x] Criar UI, métricas, runbook e E2E com receiver local.

## Verificação

Rollback não publica; commit publica uma vez; receiver duplicado deduplica;
SSRF/replay/timeout/DLQ cobertos; canários provam ausência de secrets.

## Evidência de fechamento

- Migration `00008`: catálogo/outbox append-only, targets cifrados, deliveries,
  leases, circuit state e DLQ; `events:read` permanece independente dos scopes.
- Toda mutação mapeada publica a partir do mesmo helper transacional da auditoria;
  testes provam commit único e rollback de domínio/auditoria quando outbox falha.
- Worker usa `SKIP LOCKED`, IPs aprovados fixados no dial, redirects bloqueados,
  HMAC, retry/jitter, circuito, DLQ, replay auditado e retenção fail-closed.
- Console `/integrations/webhooks` usa DataTable para targets/deliveries e cobre
  indicadores operacionais, secret one-shot, enable/disable e replay em
  pt-BR/en-US/es-ES.
- Receiver Compose recebeu evento real; Playwright recalculou a assinatura e
  conferiu `delivered`, retry-zero e limpou o tenant da fixture. A suíte completa
  passou 19/19 no produto empacotado, route smoke Vite e OIDC/Dex 2/2.
- Testes PostgreSQL cobrem retenção de deliveries/DLQ/eventos órfãos e replay
  transacional auditado, incluindo rejeição de segundo replay.
