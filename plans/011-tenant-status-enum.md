# Plan 011: Validate tenant status enum on update

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/admin_actions.go server/migrations/ server/internal/httpapi/admin_test.go`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: 002 (inactive resolve) — complementary; can land either order
- **Category**: bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`updateTenant` accepts any non-empty `status` string. Resource/definition/API
client handlers whitelist `active|inactive`. Typos (`actve`) persist, break UI
readiness checks, and confuse future resolve filters (plan 002).

## Current state

```go
// admin_actions.go updateTenant
if in.Status == "" {
  in.Status = "active"
}
// no whitelist — unlike setResourceStatus
```

Schema `tenants.status` is unconstrained text (`00001_init.sql`).

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |

## Scope

**In**:
- `updateTenant` validation
- Optional migration CHECK constraint
- Test for 400 on invalid status
- Existing `TestUpdateTenant_Persists` if present must still pass with `inactive`

**Out**:
- Multi-status workflows (pending, etc.)
- SPA form redesign

## Git workflow

- Branch: `advisor/011-tenant-status-enum`
- Commit: `fix(admin): whitelist tenant status active|inactive`
- No push/PR unless asked.

## Steps

### Step 1: Handler whitelist

```go
if in.Status == "" {
  in.Status = "active"
}
if in.Status != "active" && in.Status != "inactive" {
  writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
  return
}
```

Match message style of `setResourceStatus`.

**Verify**: build + existing tests.

### Step 2 (optional): DB CHECK

New goose migration:

```sql
-- +goose Up
UPDATE tenants SET status = 'active' WHERE status NOT IN ('active', 'inactive');
ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
  CHECK (status IN ('active', 'inactive'));
-- +goose Down
ALTER TABLE tenants DROP CONSTRAINT tenants_status_check;
```

Only if you are comfortable with migrations in this batch; handler-only is
enough for S.

### Step 3: Test

PUT tenant with `status: "nope"` → 400.
PUT `inactive` → 200 (existing).

**Verify**: `go test ./internal/httpapi/ -count=1`

## Done criteria

- [x] Invalid status rejected with 400
- [x] active/inactive still work
- [x] README `DONE`

## STOP conditions

- Existing rows have non-enum statuses in a shared env you must not rewrite —
  skip migration, do handler only.

## Maintenance notes

- Keep enum list shared if more entities grow (constants package).
