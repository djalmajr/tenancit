# História 05 — Observabilidade e saúde operacional

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

O console precisa explicar falhas de PostgreSQL, Valkey, OIDC, outbox, uso,
auditoria e jobs. Relatos do host são úteis, mas não substituem probes síncronos.

## Arquivos

- Pacote telemetry, instrumentação HTTP/DB/Valkey/workers e endpoints métricos.
- Schema/API de reports operacionais autenticados e tela `/operations/health`.
- Dashboards, alertas, SLOs e runbooks.

## Detalhe

OpenTelemetry configurável e desligado por default local, logs estruturados e
RED/USE metrics. Reports possuem origem, idempotency key, freshness e status;
stale nunca aparece como healthy. `/healthz` é liveness e `/readyz` detalha
classes sem revelar configuração interna.

## Tarefas

- [ ] Definir nomenclatura, cardinalidade e redaction policy.
- [ ] Instrumentar auth, RBAC, CSRF, limiter, DB, audit, usage e outbox.
- [ ] Implementar reports de backup/restore/rewrap/migration com credencial própria.
- [ ] Criar health UI e alertas por freshness/SLO.
- [ ] Adicionar testes canário para logs, métricas e traces.

## Verificação

Collector local recebe traces/métricas; dependência quebrada muda readiness e
alerta; nenhum token, cookie, hash, payload ou URL sensível é exportado.

