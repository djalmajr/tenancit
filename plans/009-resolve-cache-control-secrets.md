# Plan 009: Safe Cache-Control for secret-bearing resolve responses

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/consumer.go docs/developers/03-contratos-http.adoc server/internal/httpapi/integration_test.go`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW–MED (clients that relied on browser HTTP cache of resolve bodies)
- **Depends on**: none; coordinate messaging with plan 018/020 (edge path)
- **Category**: security
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`GET /v1/resolve` returns **decrypted secrets** and sets
`Cache-Control: private, max-age=30`. Browsers and some intermediaries may
retain cleartext credentials. Intended consumers are server-to-server (edge
injector + apps), which should manage their own cache via ETag. Positive
`max-age` on secret bodies is the wrong default for a secrets control plane.

`/v1/identify` returns only `tenantSlug` — short `max-age` is acceptable
(currently 300).

## Current state

```go
// consumer.go
const resolveMaxAge = "private, max-age=30"
const identifyMaxAge = "private, max-age=300"
// handleResolve sets Cache-Control + ETag; 304 skips body
```

Docs: `docs/developers/03-contratos-http.adoc` documents `private, max-age=30`.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Tests | `cd server && go test ./internal/httpapi/ -count=1 -run 'Resolve|Identify|ETag'` | exit 0 |

## Scope

**In**:
- `server/internal/httpapi/consumer.go` Cache-Control for resolve (+ resolve-one
  if headers added)
- Docs contract update
- Test assertions on header values

**Out**:
- Removing ETag / 304 support
- CDN product design
- Changing identify max-age unless clearly wrong

## Git workflow

- Branch: `advisor/009-resolve-cache-control-secrets`
- Commit: `fix(resolve): no-store Cache-Control on secret responses`
- No push/PR unless asked.

## Steps

### Step 1: Choose header policy

**Recommended**:

```go
const resolveCacheControl = "private, no-store"
```

Keep setting `ETag`. Clients that implement conditional GET store the ETag
themselves (app memory / their cache), not via shared browser HTTP cache.

If you must preserve revalidation hints without storing body:

```go
// only if you document client-managed validators:
"private, no-cache"
```

Prefer **`no-store`** for secret bodies.

Apply to:
- `handleResolve` 200 responses
- `handleResolveOne` 200 responses (today may omit Cache-Control — set the same)
- 304 responses: still set ETag + same Cache-Control

**Do not** put secrets in logs when debugging headers.

**Verify**: build + tests.

### Step 2: Update docs

`docs/developers/03-contratos-http.adoc`: replace `private, max-age=30` with
the new value; note that clients should cache using ETag in application layer.

### Step 3: Tests

In `TestE2E_ResolveByTenantIdAndETag` (or new test): assert
`Cache-Control` contains `no-store` (or chosen policy) and still has ETag;
304 path still works with `If-None-Match`.

**Verify**: integration tests pass.

## Done criteria

- [x] Resolve secret responses do not advertise positive max-age storage
- [x] ETag/304 still work in tests
- [x] Docs match code
- [x] README `DONE`

## STOP conditions

- A documented production client **requires** HTTP-cache max-age of resolve
  bodies — STOP and report before changing.
- You are tempted to remove ETag — out of scope.

## Maintenance notes

- Edge should prefer `/v1/identify` (no secrets) for long cache; resolve with
  no-store + app-level ETag.
- Reviewer: grep `max-age=30` in repo after change.
