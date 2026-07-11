# Plan 002: Reject resolve/identify for inactive tenants

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer maintains the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- server/internal/store/queries/tenants.sql server/internal/service/resolve.go server/internal/httpapi/consumer.go server/internal/httpapi/integration_test.go server/internal/store/db/tenants.sql.go`
> On mismatch with "Current state", STOP.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security | bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Operators can set a tenant’s `status` to `inactive` in the admin UI (readiness
treats active as required), but consumer endpoints still return identity and
**decrypted secrets** for that tenant. “Offboard / suspend tenant” must stop
secret delivery for any valid API client. Match unknown-tenant behavior with
`404` so clients cannot distinguish inactive vs missing (no extra probe surface).

## Current state

- `server/internal/store/queries/tenants.sql` — lookups ignore status:

```sql
-- name: GetTenantByHostname :one
SELECT t.* FROM tenants t
JOIN tenant_domains d ON d.tenant_id = t.id
WHERE d.hostname = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;
```

- `server/internal/httpapi/consumer.go` — `handleIdentify` / `handleResolve` /
  `handleResolveOne` use resolver lookups; on any lookup error → 404
  `"tenant not found"`.
- `server/internal/service/resolve.go` — `TenantByHostname` / `TenantBySlug`
  pass through to queries.
- Admin update already allows `status: "inactive"`
  (`admin_actions.go` `updateTenant`).
- Integration helpers live in `server/internal/httpapi/integration_test.go`
  (`newTestServer`, `do`, `seedTenant`, `mintToken`, `seedDefinition`).

**Product vocabulary**: tenant status is `active | inactive` (design/schema).
Consumer auth remains API client Bearer (ADR 0004) — do not change that model.

**Preferred fix location**: enforce in the **consumer handlers or resolver**
after load (`if tenant.Status != "active" { treat as not found }`), **or**
filter in SQL. Handler/service check is clearer for all three entry points
and avoids changing admin `GetTenant` behavior.

Do **not** filter `GetTenant` (admin by id) or `ListTenants` — operators must
still see inactive tenants.

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| sqlc (only if SQL queries change) | `make sqlc` | exit 0 |
| Go tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |
| Full Go | `cd server && go vet ./... && go test ./...` | exit 0 |

Docker required for testcontainers-backed HTTP tests.

## Scope

**In scope**:
- `server/internal/httpapi/consumer.go` and/or `server/internal/service/resolve.go`
- Optionally `server/internal/store/queries/tenants.sql` + regenerate sqlc
  **only if** you choose SQL filtering via **new** query names (do not change
  admin-facing GetTenant semantics accidentally)
- `server/internal/httpapi/integration_test.go` (new test)

**Out of scope**:
- SPA readiness copy changes
- Deactivating resources vs tenants (resources already filtered by
  `ListActiveResourcesByTenant`)
- API client revoke path
- Hostname case normalization (separate finding)
- Cache-Control / ETag changes

## Git workflow

- Branch: `advisor/002-reject-inactive-tenant-resolve`
- Commit message style: `fix(resolve): 404 inactive tenants on identify/resolve`
- Do NOT push/PR unless asked.

## Steps

### Step 1: Add a single “tenant consumable?” gate

Implement one helper used by all consumer paths, e.g. in `resolve.go` or
`consumer.go`:

```go
func tenantIsActive(t db.Tenant) bool {
	return t.Status == "active"
}
```

After successful `TenantByHostname` / `TenantBySlug` / tenant load inside
`ByHostnameAndDefinition`, if not active, return the same error path as
“not found” so handlers respond with **404** and body
`{"error":"tenant not found"}` (same as today for missing tenants).

**Do not** return 403 with a distinct message that reveals the tenant exists.

Cover:

1. `handleIdentify`
2. `handleResolve` (hostname and tenantId)
3. `handleResolveOne` (`ByHostnameAndDefinition` path)

**Recommended approach (minimal blast radius)**:

In `consumer.go` after each successful tenant fetch:

```go
if tenant.Status != "active" {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
	return
}
```

For `handleResolveOne`, either:

- Load tenant first and check status before resolving one resource, or
- Change `ByHostnameAndDefinition` / add a wrapper that checks status after
  `GetTenantByHostname`.

**Verify**: `cd server && go build ./...` → exit 0

### Step 2: Integration test

Add `TestE2E_InactiveTenant_NotResolvable` in
`server/internal/httpapi/integration_test.go`, modeled after
`TestE2E_AdminCreateThenResolve` / `TestE2E_Identify`:

1. `seedDefinition`, `seedTenant` with hostname, create resource with secret,
   `mintToken`.
2. Confirm resolve by hostname → 200 (active).
3. `PUT /v1/admin/tenants/{id}` with `status: "inactive"` (and valid name/slug).
4. Assert:
   - `GET /v1/resolve?hostname=...` with API key → **404**
   - `GET /v1/resolve?tenantId={slug}` → **404**
   - `GET /v1/identify?hostname=...` → **404**
   - Optional: `GET /v1/resolve/{hostname}/resources/{defKey}` → **404**
5. Assert 404 body does **not** contain secret cleartext.
6. Reactivate tenant (`status: "active"`) → resolve 200 again (optional but good).

**Verify**:
`cd server && go test ./internal/httpapi/ -count=1 -run 'TestE2E_InactiveTenant_NotResolvable|TestE2E_AdminCreateThenResolve|TestE2E_Identify' `
→ all pass.

### Step 3: Regression suite

**Verify**: `cd server && go vet ./... && go test ./...` → exit 0

## Test plan

| Case | Expected |
|------|----------|
| Active tenant resolve/identify | 200 (unchanged) |
| Inactive tenant resolve by hostname | 404, no secrets |
| Inactive tenant resolve by tenantId | 404 |
| Inactive identify | 404 |
| Inactive resolve-one | 404 |
| Admin GET tenant still works for inactive | 200 (out of path; optional assert) |

Pattern: `integration_test.go` helpers (`do`, `mintToken`, etc.).

## Done criteria

- [x] Inactive tenant cannot obtain secrets via identify/resolve/resolve-one
- [x] Response status is 404 with same public error shape as unknown tenant
- [x] New integration test exists and passes
- [x] `go test ./...` under `server/` exits 0
- [x] No out-of-scope file changes
- [x] `plans/README.md` 002 → `DONE`

## STOP conditions

- Product owner / docs explicitly require inactive tenants to keep resolving
  (none found at plan time — if you find an ADR saying that, STOP).
- Existing tests assert inactive tenants **must** resolve — STOP and report.
- sqlc regenerate rewrites unrelated generated files massively — only commit
  files required by this change.

## Maintenance notes

- Reviewer: confirm admin still lists/edits inactive tenants.
- If multi-status enums grow, centralize allowed consumer statuses in one helper.
- Pair with future audit log: “resolve denied: tenant inactive” is useful.
