# Saúde operacional e telemetria

**Status:** VALIDADO localmente com receiver OTLP/HTTP de teste, PostgreSQL e
Valkey descartáveis em 2026-07-11.

## Verificação rápida

```bash
curl -fsS http://localhost:8080/healthz | jq
curl -fsS http://localhost:8080/readyz | jq
```

`/healthz` prova apenas que o processo HTTP responde. `/readyz` executa probes
síncronos e retorna `503` se PostgreSQL ou Valkey estiverem indisponíveis. O IdP
é não crítico para tráfego de consumo e aparece como `degraded`. A resposta não
inclui DSN, URL, erro interno ou credencial.

No console, abra `/operations/health` para dependências, fila de webhooks,
circuitos e reports recentes. `stale` exige investigação mesmo se o último status
gravado era `healthy`.

## Habilitar OTLP

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT='https://otel.example.internal:4318'
export OTEL_SERVICE_NAME='tenancit-production'
```

Para collector HTTP local sem TLS, declare conscientemente:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT='http://127.0.0.1:4318'
export OTEL_EXPORTER_OTLP_INSECURE='true'
```

Configure autenticação do exporter no secret manager conforme o collector. Não
imprima `OTEL_EXPORTER_OTLP_HEADERS`. Após iniciar, gere uma request e confirme
traces e métricas `tenancit.*`; os testes automatizados fazem o mesmo contra um
receiver OTLP/HTTP.

## Publicar evidência de backup

```bash
curl -fsS -X POST http://localhost:8080/v1/operations/reports \
  -H "Authorization: Bearer $TENANCIT_OPERATIONS_REPORT_TOKEN" \
  -H "Idempotency-Key: backup-$(date -u +%F)" \
  -H 'Content-Type: application/json' \
  --data "{\"kind\":\"backup\",\"source\":\"postgres-primary\",\"status\":\"healthy\",\"occurred_at\":\"$(date -u +%FT%TZ)\",\"fresh_for_seconds\":86400}"
```

Repetir a mesma key e o mesmo payload retorna o report existente. A mesma key
com payload diferente retorna `409 idempotency_conflict`. Nunca coloque nomes de
arquivos, buckets, URLs, mensagens de erro ou secrets no report.

## Resposta

- PostgreSQL/Valkey `unavailable`: retirar a réplica do balanceamento e seguir o
  runbook da dependência.
- IdP `degraded`: preservar Consumer API, investigar discovery/TLS/DNS e impedir
  novos logins até recuperar.
- dead letters/circuito: abrir Integrações, corrigir receiver e usar replay.
- report `failed`/`stale`: bloquear mudança dependente (deploy, rewrap ou restore)
  até nova evidência válida.
