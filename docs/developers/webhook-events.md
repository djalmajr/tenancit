# Webhooks e change feed

## Envelope v1

Cada mutação suportada grava auditoria, evento e deliveries na mesma transação.
O envelope contém ID, tipo `tenancit.*`, versão, instante, request ID, aggregate
type/ID e `data` apenas com referências. Nome, hostname, valores, URL, token,
hash e secrets nunca entram.

Catálogo v1: `tenant.created|updated|deleted`, `tenant_domain.added|deleted`,
`resource_definition.created|status_changed`, `resource_field.added|deleted`,
`tenant_resource.provisioned|status_changed|deleted` e
`api_client.created|policy_updated|rotated|revoked|deleted`.

## Assinatura e replay

Headers: `Tenancit-Webhook-Id`, `Tenancit-Webhook-Timestamp` (Unix seconds) e
`Tenancit-Webhook-Signature`. A assinatura é
`v1=hex(HMAC-SHA256(secret, timestamp + "." + raw_body))`. O receiver deve
rejeitar timestamps fora de cinco minutos, verificar HMAC em tempo constante e
deduplicar por ID. Retries preservam o mesmo event ID.

## Change feed

`GET /v1/events?cursor=&limit=` exige `events:read`, retorna no máximo 200
envelopes em ordem cronológica e cursor opaco. Não concede `tenant:identify`
nem `resource:resolve` e responde `private, no-store`.
