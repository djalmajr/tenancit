# Plan 025: Stable 409 when deleting resource fields still in use

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/admin_actions.go server/migrations/00001_init.sql`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: 004 (error hygiene), 006 (parent ownership) preferred
- **Category**: bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`tenant_resource_values.resource_field_id` references `resource_fields` without
ON DELETE CASCADE. Deleting a field that still has values fails with FK
violation; handler returns 500 + raw SQL (`err.Error()` today). Operators need
a clear 409: field is in use.

## Current state

- `deleteField` in `admin_actions.go`
- Migration FK default RESTRICT
- Plan 004 removes raw errors; this plan maps FK to product message

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |

## Scope

**In**: `deleteField` error mapping (+ helper `isForeignKeyViolation`); test

**Out**: Implementing cascade delete of values (product decision — if you
choose cascade, STOP and get confirmation first)

## Git workflow

- Branch: `advisor/025-field-delete-fk-conflict`
- Commit: `fix(admin): 409 when deleting field with values`
- No push/PR unless asked.

## Steps

### Step 1: Map FK violation

```go
if isForeignKeyViolation(err) {
  writeJSON(w, http.StatusConflict, map[string]string{
    "error": "field is in use by tenant resources",
  })
  return
}
```

Use `pgconn.PgError` code `23503`.

### Step 2: Test

Create def+field+tenant resource with value; DELETE field → 409 stable
message; field still listed.

**Verify**: tests pass.

## Done criteria

- [x] In-use field delete → 409, no SQL text
- [x] Unused field delete → 204 (with 006 parent checks)
- [x] README `DONE`

## STOP conditions

- Product wants cascade delete of values — separate explicit decision.

## Maintenance notes

- SPA can map 409 to i18n via `apiErrorMessage` later.
