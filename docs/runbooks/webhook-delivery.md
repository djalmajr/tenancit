# Runbook — entrega de webhooks

- Targets aceitam HTTPS público. HTTP/local exige `TENANCIT_DEV_MODE=true` e
  `TENANCIT_WEBHOOK_ALLOW_LOOPBACK_HTTP=true` para fixtures descartáveis.
- URL e signing secret ficam cifrados; o secret aparece uma vez.
- Worker: lease/`SKIP LOCKED`, timeout 10 s, redirects bloqueados, DNS/IP fixado.
- `408`, `429`, `5xx` e rede recebem retry exponencial com jitter; `4xx`
  permanente ou oito tentativas vão à DLQ. Cinco falhas abrem circuito por 1 min.
- Corrija o receiver e use replay; a ação exige `integration.manage` e é auditada.
- `webhook_delivery_retention_days` e `outbox_event_retention_days` governam
  limpeza. Falha ao ler settings interrompe o job sem apagar.

Nunca copie body, URL completa ou secret para logs ou tickets.
