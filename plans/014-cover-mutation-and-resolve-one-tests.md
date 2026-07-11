# Plan 014: Cover resolve-one and admin mutation endpoints in tests

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: 006 if testing ownership (write tests that match final SQL);
  otherwise can land before 006 with weaker parent assertions
- **Category**: tests
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Registered but thinly tested:

- `GET /v1/resolve/{hostname}/resources/{definitionKey}`
- `DELETE` domain / resource / field
- `PUT` api-client status (revoke → resolve 401)

Regressions on operator lifecycle and single-resource resolve ship easily.

## Current state

- Helpers: `integration_test.go` — `newTestServer`, `do`, `seedDefinition`,
  `seedTenant`, `mintToken`
- E2E resolve/ETag/identify exist; resolve-one and deletes lack primary asserts
- Pattern: table-driven status checks + `mustJSON`

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |

## Scope

**In**: `server/internal/httpapi/*_test.go` only (prefer no production changes)

**Out**: Playwright E2E, SPA route tests (plan 015/elsewhere), service unit
tests (plan 015)

## Git workflow

- Branch: `advisor/014-cover-mutation-and-resolve-one-tests`
- Commit: `test(httpapi): cover resolve-one, deletes, api client revoke`
- No push/PR unless asked.

## Steps

### Step 1: Resolve-one

Happy path: seed def+tenant+resource+token; GET resolve-one → 200; password
decrypted; host present.
Missing resource → 404.
Missing tenant → 404.
No API key → 401.

### Step 2: Deletes

- Domain delete → 204; list domains empty
- Resource delete → 204; resolve no longer includes it
- Field delete on unused field → 204
- Field delete when values exist → document current behavior (500/409); if plan
  004 landed, expect stable conflict message — assert without requiring SQL text

### Step 3: API client revoke

Create client; resolve 200; set status inactive; resolve 401; reactivate;
resolve 200 (if product allows reactivate — UI does).

**Verify**: full `go test ./internal/httpapi/ -count=1`

## Done criteria

- [x] resolve-one covered
- [x] domain/resource delete covered
- [x] api client revoke affects resolve
- [x] README `DONE`

## STOP conditions

- Product changes mid-flight break seed helpers — fix tests only unless bug
  found; if production bug, STOP and open/point to fix plan (002–006).

## Maintenance notes

- Keep tests mutation-style comments like existing suite.
