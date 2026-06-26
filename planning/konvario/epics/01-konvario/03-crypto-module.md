# Story 3 — crypto-module

**Origin:** `planning/konvario/epics/01-konvario/00-overview.md`
**Status:** [x] Done

## Context
- **Objetivo:** AES-256-GCM com chave externalizada e `key_version` para rotação.
- **Refs:** decisão #4/#5; `business/rules` RN-04/RN-05.

## Traceability
- Business rules: RN-04 (cripto de secrets), RN-05 (decrypt server-side).

## Files
| File | Action | Reason | Confidence |
|---|---|---|---|
| `server/internal/crypto/crypto.go` | Create | Cryptor AES-256-GCM versionado | core |
| `server/internal/crypto/config.go` | Create | `FromEnv` (KONVARIO_AES_KEY base64 + versão) | core |
| `server/internal/crypto/crypto_test.go` | Create | round-trip, tamper, nonce, key size, cross-version | core |

## Detail
### TO-BE
- `Encrypt(plaintext) -> {Cipher, Nonce, KeyVersion}`; `Decrypt` usa a versão registrada.
- Chave de env; erro claro se ausente ou != 32 bytes; nonce aleatório por operação.

## Acceptance criteria
- [x] Round-trip correto; ciphertext não vaza plaintext.
- [x] GCM detecta adulteração (tamper → erro).
- [x] Nonce difere entre operações; chave != 32 bytes rejeitada.
- [x] Decrypt cross-version (dado v1 abre com Cryptor v2 que conhece v1).

## Test-first plan
- **Behavior:** cifrar/decifrar round-trip + autenticação (tamper falha).
- **First failing test:** `crypto_test.go` sem implementação (Red).
- **Level:** unit.

## Tasks
- [x] **Red:** testes de round-trip/tamper/nonce/keysize/cross-version.
- [x] **Green:** `crypto.go` + `config.go`.
- [x] **Refactor:** vet limpo.

## Verification
- [x] `go test ./internal/crypto/` — verde.
