# ADR 0006 — Valkey para rate limit global de API clients

- **Status:** Aceito
- **Data:** 2026-07-10

## Contexto

O Tenancit pode operar com mais de uma réplica. Um bucket em memória por
processo multiplica o limite pelo número de réplicas e perde estado em restart.
PostgreSQL poderia serializar a decisão, mas colocaria contenção de alta
frequência no banco operacional.

## Decisão

O rate limit de produção será implementado sobre Valkey/Redis por uma interface
`RateLimiter`, usando operação atômica, TTL de buckets inativos e chaves
derivadas somente do ID não secreto do API client.

O check acontece depois de autenticação e scope e antes do handler. Falha do
Valkey retorna `503 rate_limiter_unavailable`; não existe fallback silencioso
para buckets locais. Uma implementação em memória pode ser habilitada apenas
em desenvolvimento explícito de instância única e deve emitir aviso no boot.

## Consequências

- Compose local e E2E ganham um serviço Valkey; produção fornece endpoint com
  autenticação/TLS conforme o ambiente.
- Readiness e métricas distinguem indisponibilidade e latência do limiter.
- Testes com duas instâncias precisam provar limite combinado e restart.
- Tokens e hashes nunca aparecem em keys, logs ou métricas do Valkey.

## Alternativas rejeitadas

- Bucket local: não preserva o limite global.
- PostgreSQL no hot path: aumenta contenção no banco de domínio.
- Fail-open: remove o controle justamente durante falhas.
