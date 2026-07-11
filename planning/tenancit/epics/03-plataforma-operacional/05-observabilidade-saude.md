# História 05 — Observabilidade e saúde operacional

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

O console precisa explicar falhas de PostgreSQL, Valkey, OIDC, outbox, uso,
auditoria e jobs. Relatos do host são úteis, mas não substituem probes síncronos.

## Responsabilidade, motivação e valor

Esta história observa **o próprio Tenancit e as dependências necessárias para
ele identificar tenants e entregar configurações**. Ela responde se o banco do
Tenancit, seu Valkey, IdP, filas e workers estão operáveis, reduzindo falhas
silenciosas e tempo de diagnóstico.

Não testa se cada PostgreSQL, Redis, Kafka ou outro recurso cadastrado por um
tenant está disponível. Esses registros são configurações entregues aos
aplicativos. Monitorá-los exigiria uma capacidade futura e opt-in de
`Resource Health`, com probes, credenciais, scheduler, histórico e alertas
próprios; isso não será introduzido implicitamente.

**Ganho:** saber se o Tenancit consegue cumprir sua função central, sem
transformá-lo em substituto do monitoramento das aplicações clientes.

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

- [x] Definir nomenclatura, cardinalidade e redaction policy.
- [x] Instrumentar auth, RBAC, CSRF, limiter, DB, audit, usage e outbox.
- [x] Implementar reports de backup/restore/rewrap/migration com credencial própria.
- [x] Criar health UI e alertas por freshness/SLO.
- [x] Adicionar testes canário para logs, métricas e traces.

## Verificação

Collector local recebe traces/métricas; dependência quebrada muda readiness e
alerta; nenhum token, cookie, hash, payload ou URL sensível é exportado.

## Evidência de fechamento

- `internal/telemetry` configura OTLP/HTTP somente com endpoint explícito,
  exporta traces/métricas e aplica nomes/dimensões allowlisted; receiver real de
  teste confirma `/v1/traces` e `/v1/metrics` no shutdown/flush.
- Middleware HTTP usa route template, tracer PGX nunca exporta SQL/args e sinais
  fechados cobrem auth, scope, RBAC, CSRF, limiter, Valkey, audit, usage e workers
  outbox/retention. Handler JSON converte errors em `error_type` e redige chaves
  sensíveis.
- `/healthz` permanece liveness; `/readyz` faz probes síncronos de PostgreSQL,
  Valkey e IdP, responde `503` para componente crítico e nunca serializa erro,
  DSN ou URL.
- Migration `00009` e `POST /v1/operations/reports` entregam reports append-only
  com credencial exclusiva, idempotency conflict estável e freshness; o console
  `/operations/health` mostra dependências, filas, circuitos e `stale` em três
  idiomas.
- Gates: Go estrito, 19 arquivos/79 testes web, bundle abaixo do budget,
  Playwright empacotado 20/20 + route smoke Vite e OIDC/Dex 2/2. Browser confirmou
  estado geral saudável, PostgreSQL/Valkey e zero dead letters no dev local.
