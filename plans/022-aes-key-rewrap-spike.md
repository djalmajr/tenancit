# Plan 022: Spike design — AES key rewrap job (roadmap P2)

> **Executor instructions**: **Design/spike only** unless operator expands
> scope. Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/crypto/ docs/adr/0003-secrets-server-side-aes-gcm.md docs/business/04-escopo-e-roadmap.adoc docs/developers/04-seguranca-e-criptografia.adoc`

## Status

- **Status**: DONE — spike documental; nenhum job foi implementado
- **Priority**: P3 (roadmap P2)
- **Effort**: L full; **spike S–M**
- **Risk**: HIGH if naive rewrap in prod
- **Depends on**: none
- **Category**: direction | security
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Schema stores `key_version`; crypto loads `TENANCIT_AES_KEY_V<n>` for decrypt.
There is **no** job/endpoint to re-encrypt rows under a new current version.
Operators cannot retire a compromised key without offline undocumented work.

## Current state

- `crypto.FromEnv`, `Encrypt`/`Decrypt` with versioned keys
- Values table: `value_cipher`, `nonce`, `key_version`
- ADR 0003 + roadmap: rotation flow incomplete

## Scope

**In**: design for batch rewrap, safety checks, ops runbook outline;
optional dry-run CLI sketch under `server/cmd/` **only if** tiny and offline

**Out**: Automatic rewrap on every request; HSM integration; multi-region

## Git workflow

- Branch: `advisor/022-aes-key-rewrap-spike`
- Commit: `docs: AES key rewrap design spike`
- No push/PR unless asked.

## Steps

### Step 1: Document algorithm

For each `tenant_resource_values` row with cipher:

1. Decrypt with `key_version`
2. Encrypt with current version
3. Update cipher, nonce, key_version in a transaction batch
4. Verify sample decrypt before dropping old env keys

### Step 2: Failure modes

- Missing historical key at boot (fail-closed options)
- Concurrent provision during rewrap
- Partial batch crash recovery (idempotent rewrap)
- Metrics: rows remaining on old versions

### Step 3: Interface options

- Offline `go run ./cmd/rewrap` with DSN + env keys
- Admin-triggered job later

Design registrado em `docs/developers/design/aes-key-rewrap.md`; procedimento
operacional futuro em `docs/runbooks/aes-key-rewrap.md`.

**Never** put real key material in the design doc.

## Done criteria

- [x] Design + runbook outline registrados
- [x] Riscos e idempotência tratados no desenho
- [x] Plano marcado `DONE` (índice atualizado separadamente pelo coordenador)

## STOP conditions

- Implementing prod rewrap without dry-run/backup plan — STOP.

## Maintenance notes

- Pair with plan 017 FromEnv tests before automation.
