# ADR 0003 — Secrets cifrados com AES-GCM e decrypt server-side

- **Status:** Aceito
- **Data:** 2026-06-23

## Contexto

Vários resources contêm secrets operacionais, como senhas de banco e tokens. O
serviço consumidor precisa receber esses valores prontos para uso, mas a chave
de criptografia não deve sair do KonvarIO.

As alternativas consideradas foram:

- guardar secrets em texto puro;
- delegar decrypt para consumidores;
- usar secret manager externo como dependência obrigatória do hot path;
- cifrar no serviço e descriptografar server-side.

## Decisão

Campos marcados como `is_secret` são cifrados em repouso com AES-256-GCM. A
chave atual vem de `KONVARIO_AES_KEY` e a versão de `KONVARIO_AES_KEY_VERSION`. O schema
guarda `key_version` para permitir rotação futura.

O decrypt acontece server-side:

- consumers autenticados recebem cleartext sobre TLS em `/v1/resolve`;
- admin recebe secrets mascarados por padrão;
- `?reveal=true` só deve ser usado em fluxo admin autenticado e explícito.

## Consequências

Positivas:

- consumidores não lidam com criptografia;
- chave fica restrita ao serviço;
- GCM detecta adulteração;
- schema já suporta versões de chave.

Custos:

- perda de chave torna dados irrecuperáveis;
- reveal admin precisa ser tratado como operação sensível;
- rotação automática ainda requer fluxo operacional dedicado.

## Status

Aceito em 2026-06-23. Fonte de verdade: `server/internal/crypto/` e
`server/internal/service/values.go`.
