# Plan 008: Make resolve ETag reflect value and definition-field changes

> **Executor instructions**: Follow step by step; verify each step; STOP on
> conditions below. Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/service/resolve.go server/internal/store/queries/resources.sql server/internal/httpapi/integration_test.go server/internal/httpapi/consumer.go`

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: none (pair with any future “edit resource values” feature)
- **Category**: bug | correctness
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`/v1/resolve` uses strong ETags + `If-None-Match` → 304 without re-decrypt.
`computeETag` only hashes tenant + each active resource’s id/status/`UpdatedAt`.
`UpsertResourceValue` does **not** bump `tenant_resources.updated_at`. Adding
definition fields also changes the resolve payload without touching resource
rows. Today there is no admin “edit values” API, so production impact is
latent — but docs claim “any update” flips the ETag, and the next value-edit
feature will silently ship stale secrets to consumers.

## Current state

`resolve.go` `computeETag`:

```go
fmt.Fprintf(h, "t:%s:%d:%d\n", t.ID, t.UpdatedAt.UnixNano(), len(sorted))
for _, res := range sorted {
  fmt.Fprintf(h, "r:%s:%s:%d\n", res.ID, res.Status, res.UpdatedAt.UnixNano())
}
```

`resources.sql` `UpsertResourceValue` — updates value columns only.

Docs: `docs/developers/03-contratos-http.adoc` — “Qualquer alteração em tenant
ou resource (add/remove/update/status) muda o ETag”.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| sqlc | `make sqlc` | exit 0 if SQL changes |
| Tests | `cd server && go test ./internal/httpapi/ ./internal/service/ -count=1` | exit 0 |

## Scope

**In**:
- ETag computation and/or SQL that bumps resource `updated_at` on value upsert
- Optional: include definition field set fingerprint in ETag
- Tests proving ETag changes when values change
- Docs line in `03-contratos-http.adoc` if behavior description needs tweak

**Out**:
- Building a full public “PATCH resource values” product UI (may add a **test-only**
  or admin upsert path if none exists — prefer fixing upsert SQL used by
  provision so any future upsert is correct)
- Changing Cache-Control (plan 009)
- Batch N+1 rewrite (plan 012)

## Git workflow

- Branch: `advisor/008-etag-covers-config-mutations`
- Commit: `fix(resolve): bump ETag when resource values change`
- No push/PR unless asked.

## Steps

### Step 1: Bump parent resource timestamp on value write (preferred)

In `resources.sql`, change upsert to also touch parent, **or** add a second
query in the same provision transaction:

```sql
-- name: TouchTenantResource :exec
UPDATE tenant_resources SET updated_at = now() WHERE id = $1;
```

Call `TouchTenantResource` after successful value upserts in
`provisionResource` (once per resource after the field loop is enough).

Alternatively embed in upsert via SQL CTE — keep it explicit and testable.

**Verify**: `make sqlc` if needed; `go build ./...`

### Step 2: Cover definition field schema drift (minimum)

If a new field is added to a definition used by active resources, ETag should
change. Options (pick simplest that works):

**A.** Include in `computeETag` a hash of definition ids + field counts/keys
fetched once per resolve Version path (extra queries — may conflict with 012).

**B.** On `AddField` / `RemoveField` / definition status, `UPDATE tenant_resources
SET updated_at = now() WHERE resource_definition_id = $def` so all instances
flip.

Prefer **B** for S effort: in `addField` / `deleteField` handlers (or service),
after successful field mutation, touch resources of that definition.

**Verify**: build green.

### Step 3: Tests

1. Provision resource; capture ETag from resolve.
2. Without plan 008 there is no public value update — use either:
   - direct DB/test helper calling `UpsertResourceValue` + touch, or
   - re-provision path if available, or
   - call internal service in a unit test with fake querier asserting
     `computeETag` differs when `UpdatedAt` changes.
3. Best integration approach: after create, run SQL via test pool to upsert a
   value **without** touch → document that without touch ETag stale; with
   touch after upsert, ETag must change. Easier: unit-test `computeETag`
   with two resources differing only in `UpdatedAt`.
4. Add field to definition; assert resolve ETag changes (integration).

Existing `TestE2E_ResolveByTenantIdAndETag` must still pass (status flip).

**Verify**: `cd server && go test ./internal/httpapi/ -count=1 -run ETag` → pass

### Step 4 (optional): ETag on resolve-one

`handleResolveOne` has no ETag. Mirror bulk resolve with resource-level ETag
if cheap; otherwise leave to a follow-up note in Maintenance.

## Done criteria

- [x] Value mutation path bumps ETag inputs
- [x] Definition field add/remove flips ETag for affected tenants’ resolve
- [x] Tests lock behavior
- [x] README `DONE`

## STOP conditions

- Touching all resources on every field add is too expensive for known huge
  fleets — report; propose definition version column instead.
- computeETag rewrite requires decrypting values — **do not** decrypt for ETag.

## Maintenance notes

- Any future `PATCH` values API must go through the same touch/upsert path.
- Reviewer: 304 must not return after secret rotation/value change.
