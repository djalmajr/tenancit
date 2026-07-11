# Plan 004: Harden HTTP errors — no raw `err.Error()`, correct status codes

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in STOP conditions occurs, stop and report.
> When done, update `plans/README.md` unless the reviewer maintains the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- server/internal/httpapi/ server/internal/service/provision.go`
> On mismatch with "Current state", STOP.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: 001 recommended (not required)
- **Category**: security | bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Many handlers return `err.Error()` in JSON to clients (admin and consumer).
That leaks Postgres constraint text, driver details, and crypto failures to
anyone with a token — and the SPA surfaces `ApiError.serverMessage`. Separately,
several handlers map **any** DB error to a specific 409 message (e.g.
`setResourceStatus` always claims “active resource already exists”;
`CreateTenant` / `AddTenantDomain` / `ProvisionResource` collapse all failures
into uniqueness conflicts). Operators and clients cannot trust status codes.

## Current state

`writeJSON` helper: `server/internal/httpapi/health.go`.

Widespread pattern (examples):

```go
// admin.go listTenants
writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})

// admin_actions.go setResourceStatus
if err != nil {
  writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
  return
}

// provision.go CreateTenantResource
if err != nil {
  return db.TenantResource{}, ErrActiveResourceExists
}

// admin.go addDomain
writeJSON(w, http.StatusConflict, map[string]string{"error": "this hostname is already mapped to a tenant"})
```

**Good exemplar** (stable messages): `writeProvisionError` in `admin.go` and
tenant create’s `"a tenant with this slug already exists"`.

**SPA**: `web/src/lib/api.ts` `ApiError` + `web/src/lib/i18n.tsx`
`apiErrorMessage` — prefer stable English server messages; UI localizes by
status when possible. Do not break known clean messages already mapped.

**Never put secret values or full DSNs into logs or responses.**

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| Go tests | `cd server && go test ./internal/httpapi/ ./internal/service/ -count=1` | exit 0 |
| Full Go | `cd server && go vet ./... && go test ./...` | exit 0 |
| Grep residual | `rg 'err\\.Error\\(\\)' server/internal/httpapi --glob '*.go'` | no matches in handler responses (tests may mention strings) |

## Scope

**In scope**:
- `server/internal/httpapi/*.go` (handlers + small shared helpers; not generated)
- `server/internal/service/provision.go` (error classification only)
- Tests under `server/internal/httpapi/*_test.go` and optionally
  `server/internal/service/*_test.go`

**Out of scope**:
- Changing success JSON shapes of resources/tenants
- Full migration of admin CRUD into `service` package
- Parent ownership WHERE clauses (plan 006)
- Frontend i18n redesign
- Logging stack / OpenTelemetry

## Git workflow

- Branch: `advisor/004-harden-http-errors`
- Commits: `fix(httpapi): stop leaking err.Error to clients` and/or
  `fix(httpapi): map pg errors to correct status codes`
- Do NOT push/PR unless asked.

## Steps

### Step 1: Add a small internal error helper

In `server/internal/httpapi/` (e.g. `errors.go` or next to `writeJSON`):

```go
func writeInternalError(w http.ResponseWriter, err error) {
  // log with slog if desired: slog.Error("request failed", "err", err)
  writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
```

Optional helpers:

```go
func writeNotFound(w http.ResponseWriter) {
  writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}
```

Use `log/slog` for server-side full error (import if not present). Do **not**
log request Authorization headers.

**Verify**: `cd server && go build ./...` → exit 0

### Step 2: Replace client-facing `err.Error()` on 5xx

In all `httpapi` handlers, replace:

```go
writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
```

with `writeInternalError(w, err)`.

Also fix consumer path:

```go
// consumer.go handleResolve — Version / ResolveTenant failures
writeInternalError(w, err) // not err.Error()
```

For **409 Conflict** paths that currently use `err.Error()` (e.g. some
`updateTenant` / `addField` paths), replace with a stable message or classified
mapping from Step 3 — never raw driver text.

**Verify**:
`rg 'err\\.Error\\(\\)' server/internal/httpapi --glob '*.go' | rg -v '_test\\.go'`
→ no handler should still embed `err.Error()` in `writeJSON` payloads.

(Allow `err.Error()` only in tests or slog attributes.)

### Step 3: Classify DB errors for conflict vs not-found vs internal

Use `errors.Is(err, pgx.ErrNoRows)` and optionally `pgconn.PgError` codes:

| Code / condition | HTTP | Public message |
|------------------|------|----------------|
| `pgx.ErrNoRows` on :one updates | 404 | `not found` |
| Unique violation `23505` | 409 | existing product messages (slug/hostname/active resource/def key) |
| FK violation `23503` | 400 or 404 | e.g. `not found` or `invalid reference` |
| Other | 500 | `internal error` |

**Critical fix — `setResourceStatus`** (`admin_actions.go`):

```go
res, err := s.Q.SetTenantResourceStatus(...)
if err != nil {
  if errors.Is(err, pgx.ErrNoRows) {
    writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
    return
  }
  // unique partial index collision on re-activate → 409 known message
  if isUniqueViolation(err) {
    writeJSON(w, http.StatusConflict, map[string]string{
      "error": "an active resource of this type already exists",
    })
    return
  }
  writeInternalError(w, err)
  return
}
```

**`provision.go`**: only map to `ErrActiveResourceExists` when unique violation
(`23505`); other errors return wrapped real error so HTTP layer can 500.

**`createTenant` / `addDomain` / `createDefinition`**: only return the friendly
409 string when unique violation; otherwise `writeInternalError`.

Helper sketch:

```go
func isUniqueViolation(err error) bool {
  var pgErr *pgconn.PgError
  return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
```

Import: `github.com/jackc/pgx/v5/pgconn` and `github.com/jackc/pgx/v5`.

**Verify**: `cd server && go build ./...` → exit 0

### Step 4: Fix resolve-one error collapsing (related)

`handleResolveOne` currently maps **any** error to 404 `"tenant not found"`.
Change to:

- not found (no rows / missing) → 404 appropriate message
  (`tenant not found` vs `resource not found` as today for `found==false`)
- other errors → `writeInternalError`

**Verify**: build still green.

### Step 5: Tests

Extend `admin_test.go` / `integration_test.go`:

1. `setResourceStatus` with random UUID → **404** (not 409).
2. Existing duplicate-active still **409** with stable message
   (`TestCreateResource_DuplicateActive_409` remains green).
3. Optional: force internal path not required if hard to trigger.

For provision, unit test with fake querier is optional; HTTP-level is enough.

**Verify**:
`cd server && go test ./internal/httpapi/ -count=1` → exit 0
`cd server && go vet ./... && go test ./...` → exit 0

## Test plan

| Case | Expected |
|------|----------|
| Status update unknown resource id | 404, no SQL text |
| Duplicate active resource | 409, known English message |
| List/get happy paths | unchanged |
| Existing RN tests (mask, admin auth, provision) | still pass |
| 500 path | body `{"error":"internal error"}` only (if testable) |

## Done criteria

- [x] No `writeJSON(..., err.Error())` in non-test httpapi handlers
- [x] `setResourceStatus` missing id → 404
- [x] Uniqueness conflicts still 409 with stable messages
- [x] Consumer 500s do not include raw error strings
- [x] `go test ./...` in `server/` exits 0
- [x] Scope respected
- [x] `plans/README.md` 004 → `DONE`

## STOP conditions

- Classifying `PgError` requires a dependency not already in `go.mod` beyond
  pgx (pgx already provides pgconn) — do not add new HTTP frameworks.
- Changing public **success** payloads seems required — STOP; this plan is
  errors only.
- A test asserts raw Postgres text as the UX contract — STOP and report (should
  not exist after prior usability work).

## Maintenance notes

- Reviewer: sample several handlers for leftover `err.Error()` in responses.
- New handlers must use `writeInternalError` + domain-specific 4xx only.
- Plan 006 should return 404 with stable `"not found"` for ownership misses.
