# Observabilidade e política de redaction

## Contrato

O Tenancit exporta traces e métricas via OTLP/HTTP apenas quando
`OTEL_EXPORTER_OTLP_ENDPOINT` está configurado. O modo local permanece desligado
por padrão. Endpoints HTTP exigem `OTEL_EXPORTER_OTLP_INSECURE=true`; produção
deve preferir TLS e autenticação do collector.

Nomes estáveis:

- `tenancit.http.server.requests`, `duration`, `active_requests`;
- `tenancit.security.decisions`;
- `tenancit.dependency.operations`, `duration`;
- `tenancit.worker.cycles`, `items`, `duration`; workers `audit_export` e
  `audit_retention` cobrem materialização/expiração, partições e legal holds;
- `tenancit.rewrap.batches`, `rows.by_version`, `rows.remaining.by_version`,
  `failures`, `campaigns` e durações de lote/campanha.

Dimensões aceitas são fechadas e de baixa cardinalidade: método, route template,
status HTTP, boundary, outcome, component, operation e worker. Valores fora das
allowlists viram `other`.
No comando offline, versões numéricas de origem/alvo e classes fechadas de falha
são permitidas; `job_id` e IDs de linha não entram nas métricas.

## Dados proibidos

Logs, métricas e traces nunca recebem:

- URL completa, hostname de tenant, query string ou path parameter materializado;
- `Authorization`, cookie, CSRF, API key, signing secret, hash ou nonce;
- request/response body, payload de outbox/webhook ou configuração cifrada;
- SQL, argumentos SQL, DSN, issuer URL, endpoint de webhook ou OTLP headers.

O middleware usa apenas o template Chi (`/tenants/{id}`), e o tracer PGX registra
somente `query`, resultado e duração. Testes canário falham caso tokens, queries,
cookies, SQL ou argumentos apareçam nos atributos exportados.

## Sinais e SLO inicial

- disponibilidade da Consumer API: 99,9% mensal, excluindo `4xx` do cliente;
- p95 de `identify` e `resolve`: até 250 ms na borda da aplicação;
- disponibilidade de PostgreSQL e Valkey: readiness crítico;
- IdP: componente degradado (não derruba tráfego de API client já autenticado);
- dead letter ou circuito aberto: saúde operacional degradada;
- report expirado: sempre `stale`, nunca `healthy`.

Alertas iniciais: readiness indisponível por 2 minutos, erro `5xx` acima de 2%
por 5 minutos, dead letter não zerada por 10 minutos, circuito aberto, ausência
de backup dentro da freshness declarada e falha de worker repetida.

## Reports operacionais

Agentes de backup, restore, migration e rewrap usam exclusivamente
`POST /v1/operations/reports`, com token próprio e `Idempotency-Key`. A credencial
é configurada por `TENANCIT_OPERATIONS_REPORT_TOKEN` (mínimo 32 caracteres) e
`TENANCIT_OPERATIONS_REPORT_CREDENTIAL_VERSION`. Não é uma API key de consumo
nem sessão administrativa. Reports são append-only, não possuem metadata livre e
são classificados por freshness no momento da leitura.
