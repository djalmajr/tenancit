# Idempotência administrativa

## Escopo

O contrato protege apenas mutações cujo retry poderia duplicar entidades,
resources ou tokens: create tenant, provision resource, create API client e
rotate API client. Não é um workflow engine nem se aplica a leituras/edições
simples.

## Operação

- O cliente gera um UUID por intenção e preserva a chave enquanto o payload não
  mudar. Editar o formulário cria uma chave nova.
- Retry com mesmo principal, operação, chave e payload recebe a resposta
  original e `Idempotency-Replayed: true`.
- Reuso com payload divergente retorna `409 idempotency_mismatch`.
- Envelopes de token expiram em 10 minutos; tenant/resource em 24 horas.
- `/tenancit-audit-jobs` remove expirados em lotes de até 1.000. O role runtime
  não pode executar o cleanup nem apagar claims.

## Incidente

1. Em timeout, repetir exatamente body, principal e `Idempotency-Key`.
2. Em `idempotency_mismatch`, não force a chave: confirme a intenção e gere
   outra somente para um payload realmente novo.
3. Em `idempotency_expired`, inventarie o domínio antes de iniciar nova intenção.
4. Alertar quando `idempotency_cleanup` falhar repetidamente ou a cardinalidade
   de `admin_idempotency_records` crescer apesar dos TTLs.

Nunca copie token, body, ciphertext, nonce ou hash para logs/tickets. Evidência
permitida: request ID, operação, código estável e contagem de registros.
