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

- [ ] Versionar catálogo e schema dos eventos.
- [ ] Implementar outbox em todas as mutações relevantes.
- [ ] Implementar targets com URL cifrada, allowlist de schemes e SSRF defense.
- [ ] Assinar timestamp/body e documentar replay window.
- [ ] Implementar worker, retry, DLQ, replay autorizado e retenção.
- [ ] Expor change feed cursor-based para consumidores autorizados.
- [ ] Criar UI, métricas, runbook e E2E com receiver local.

## Verificação

Rollback não publica; commit publica uma vez; receiver duplicado deduplica;
SSRF/replay/timeout/DLQ cobertos; canários provam ausência de secrets.

