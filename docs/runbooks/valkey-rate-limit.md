# Valkey e rate limit global

**Status:** VALIDADO em Compose local e teste automatizado com duas instâncias.

## Pré-condições

- Valkey/Redis acessível com TLS e autenticação no ambiente de produção;
- `TENANCIT_VALKEY_URL` entregue pelo secret manager;
- memória dimensionada para buckets por API client e política de eviction que
  não descarte chaves ativas silenciosamente.

## Verificação

1. Execute `docker compose up -d valkey` e confira `docker compose exec valkey valkey-cli ping`.
2. Inicie a API com `TENANCIT_RATE_LIMIT_MODE=valkey`.
3. Crie um client de RPM baixo e confirme `429 rate_limited`, `Retry-After` e
   `RateLimit-*` após esgotar o bucket.
4. Execute `go test ./internal/ratelimit -run Valkey -count=1`; o teste alterna
   duas instâncias e reabre uma conexão sem zerar o bucket.

Keys contêm somente o UUID do API client. Nunca inclua token, hash, hostname ou
tenant. Latência, timeouts, recusas e erros devem ser monitorados sem labels de
alta cardinalidade.

## Indisponibilidade e recuperação

A API falha fechado com `503 rate_limiter_unavailable`; não habilite fallback
local. Restaure rede/TLS/auth do Valkey e repita a chamada. O modo `memory` é
permitido apenas em desenvolvimento explícito, uma réplica, com aviso no boot.

Rotacione a credencial do Valkey no secret manager, atualize a URL e faça rollout
das réplicas. A persistência do Valkey é uma decisão operacional: perder buckets
reinicia janelas, mas não perde autenticação, auditoria ou agregados PostgreSQL.
