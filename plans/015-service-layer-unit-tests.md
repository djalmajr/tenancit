# Plan 015: Unit tests for service core (resolve ETag, provision, apikey)

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/service/`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`server/internal/service/` holds provision rules, ETag computation, API key
hashing, and field assembly, but only `values_test.go` exists. Refactors
(plan 012) and ETag work (008) need fast offline tests without Docker.

## Current state

- Files: `resolve.go`, `provision.go`, `apikey.go`, `resource_fields.go`,
  `values.go` + `values_test.go`
- Interfaces already exist: `ResolveQuerier`, `ResourceTransactor`,
  `ResourceFieldQuerier`

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Unit | `cd server && go test ./internal/service/ -count=1` | exit 0 |
| No Docker required for pure fakes | same | no skip for these tests |

## Scope

**In**: new `*_test.go` under `server/internal/service/`

**Out**: httpapi tests (014), crypto env tests (017), changing production APIs

## Git workflow

- Branch: `advisor/015-service-layer-unit-tests`
- Commit: `test(service): unit tests for etag, provision, apikey`
- No push/PR unless asked.

## Steps

### Step 1: apikey

- `HashAPIKey` deterministic; different inputs differ; hex length 64
- `GenerateAPIToken` prefix `tnc_`; two calls differ

### Step 2: computeETag / IdentityETag

- Order of resources does not change ETag (shuffle slice)
- Changing `UpdatedAt` or status changes ETag
- IdentityETag changes when slug/updated_at changes

### Step 3: provision with fake querier/tx

Table-test:

- missing required field → `MissingRequiredFieldError`
- inactive definition → `ErrInactiveDefinition`
- unknown definition → `ErrUnknownDefinition`
- happy path calls encrypt for secret fields (use real small cryptor like
  httpapi tests)

Keep fakes small; do not require Postgres.

**Verify**: `go test ./internal/service/ -count=1`

## Done criteria

- [x] Service package tests pass offline
- [x] ETag order-independence locked
- [x] Provision error cases locked
- [x] README `DONE`

## STOP conditions

- Interfaces force full sqlc types that are painful — use minimal stubs; do not
  rewrite production interfaces unless necessary (then STOP and report).

## Maintenance notes

- Prefer these tests before large resolve refactors (012).
