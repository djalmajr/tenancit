# Plan 006: Enforce parent ownership on nested admin mutations

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in STOP conditions occurs, stop and report.
> When done, update `plans/README.md` unless the reviewer maintains the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- server/internal/httpapi/admin_actions.go server/internal/store/queries/ server/internal/store/db/ server/internal/httpapi/server.go`
> On mismatch with "Current state", STOP.

## Status

- **Priority**: P1
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: 004 optional (use stable `"not found"` / no `err.Error()`)
- **Category**: bug | security (API integrity)
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Nested routes advertise hierarchy (`/tenants/{id}/domains/{domainId}`,
`.../resources/{resourceId}`, `/resource-definitions/{id}/fields/{fieldId}`)
but handlers only use the child id. A client can DELETE/PUT a child UUID under
the wrong parent path and still mutate the object. Shared admin token means
this is not multi-user IDOR today, but path invariants are false, automation
is footgun-prone, and future finer-grained auth will inherit a broken model.
Also, deletes use `:exec` and always return **204** even when zero rows
matched (missing id).

## Current state

Routes (`server/internal/httpapi/server.go`):

```go
ar.Delete("/tenants/{id}/domains/{domainId}", s.deleteDomain)
ar.Put("/tenants/{id}/resources/{resourceId}/status", s.setResourceStatus)
ar.Delete("/tenants/{id}/resources/{resourceId}", s.deleteResource)
ar.Delete("/resource-definitions/{id}/fields/{fieldId}", s.deleteField)
```

Handlers (`admin_actions.go`) — e.g. deleteDomain:

```go
did, err := parseParam(r, "domainId")
// ... never reads tenant {id}
if err := s.Q.RemoveTenantDomain(r.Context(), did); err != nil { ... }
w.WriteHeader(http.StatusNoContent)
```

SQL (`tenants.sql` / `resources.sql` / `definitions.sql`):

```sql
-- name: RemoveTenantDomain :exec
DELETE FROM tenant_domains WHERE id = $1;

-- name: DeleteTenantResource :exec
DELETE FROM tenant_resources WHERE id = $1;

-- name: RemoveField :exec
DELETE FROM resource_fields WHERE id = $1;

-- name: SetTenantResourceStatus :one
UPDATE tenant_resources SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;
```

**Good pattern**: `DeleteTenant :execrows` + handler 404 when `n == 0`.

sqlc: after editing `server/internal/store/queries/*.sql`, run `make sqlc`.
Do not hand-edit `server/internal/store/db/*.sql.go` except via sqlc.

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| Regenerate sqlc | `make sqlc` | exit 0 |
| Go tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |
| Full Go | `cd server && go vet ./... && go test ./...` | exit 0 |

## Scope

**In scope**:
- `server/internal/store/queries/tenants.sql`
- `server/internal/store/queries/resources.sql`
- `server/internal/store/queries/definitions.sql`
- Generated `server/internal/store/db/*.sql.go` via `make sqlc` only
- `server/internal/httpapi/admin_actions.go` (and parse helpers if needed)
- Tests in `server/internal/httpapi/*_test.go`

**Out of scope**:
- List endpoints (already scoped)
- SPA route changes
- Broader IDOR/auth redesign
- Field delete FK-when-in-use product policy (409 message can be stable if you
  hit FK; do not invent cascade delete)

## Git workflow

- Branch: `advisor/006-nested-admin-parent-ownership`
- Commit: `fix(admin): scope nested deletes/updates by parent id`
- Do NOT push/PR unless asked.

## Steps

### Step 1: Change SQL to require parent + return row counts

Replace/adjust queries:

```sql
-- name: RemoveTenantDomain :execrows
DELETE FROM tenant_domains
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteTenantResource :execrows
DELETE FROM tenant_resources
WHERE id = $1 AND tenant_id = $2;

-- name: SetTenantResourceStatus :one
UPDATE tenant_resources SET status = $2, updated_at = now()
WHERE id = $1 AND tenant_id = $3
RETURNING *;

-- name: RemoveField :execrows
DELETE FROM resource_fields
WHERE id = $1 AND resource_definition_id = $2;
```

Note parameter order: match sqlc style used elsewhere (`$1` id, then status,
then tenant_id for update). Keep names clear in the query comments.

**Verify**: `make sqlc` → exit 0; generated methods take parent ids.

### Step 2: Update handlers to parse parent + child

Example `deleteDomain`:

```go
tenantID, err := parseID(r) // path {id}
// or parseParam(r, "id")
did, err := parseParam(r, "domainId")
n, err := s.Q.RemoveTenantDomain(ctx, db.RemoveTenantDomainParams{
  ID: did, TenantID: tenantID,
})
// if err → writeInternalError (or plan-004 style)
if n == 0 {
  writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
  return
}
w.WriteHeader(http.StatusNoContent)
```

Same for `deleteResource`, `deleteField`.

`setResourceStatus`: pass `TenantID` from path; on `pgx.ErrNoRows` → 404
(not 409). If plan 004 already landed, keep its unique-violation → 409 logic.

**Verify**: `cd server && go build ./...` → exit 0

### Step 3: Tests

Add tests in `admin_test.go` or `integration_test.go`:

1. **Happy path**: create tenant A + domain; DELETE with correct tenant id →
   204; list domains empty.
2. **Wrong parent**: create tenants A and B; domain under A; DELETE using
   tenant B’s id + domain A’s id → **404**; domain still listed under A.
3. **Unknown child**: correct tenant, random domain UUID → 404.
4. **Resource status / delete**: analogous cross-tenant resource id → 404.
5. **Field delete**: field under def A, DELETE under def B’s path → 404.

Use existing `do`, `seedTenant`, `seedDefinition` helpers.

**Verify**:
`cd server && go test ./internal/httpapi/ -count=1` → exit 0
`cd server && go vet ./... && go test ./...` → exit 0

## Test plan

| Case | Expected |
|------|----------|
| Delete domain correct parent | 204, gone |
| Delete domain wrong parent | 404, still present |
| Delete missing domain | 404 |
| setResourceStatus wrong tenant | 404 |
| deleteField wrong definition | 404 |

## Done criteria

- [x] Nested DELETE/PUT SQL includes parent id predicates
- [x] Zero rows / wrong parent → 404 stable message (no SQL text)
- [x] Cross-parent mutation tests pass
- [x] `make sqlc` applied; only intended generated files
- [x] `go test ./...` exits 0 under `server/`
- [x] Scope respected
- [x] `plans/README.md` 006 → `DONE`

## STOP conditions

- sqlc version mismatch breaks generate — use Makefile’s pinned
  `sqlc@v1.30.0` via `make sqlc` only.
- Schema lacks `tenant_id` on a table you need — it should exist on
  `tenant_domains` / `tenant_resources`; fields have `resource_definition_id`.
- Cascade or soft-delete product change seems required — out of scope.

## Maintenance notes

- Reviewer: confirm every nested mutator uses both path params.
- Future nested routes must copy this pattern.
- When plan 004 is done, all error bodies here should be stable strings only.
