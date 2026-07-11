# Plan 012: Batch resource field assembly (kill resolve N+1)

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/service/resource_fields.go server/internal/service/resolve.go server/internal/httpapi/admin_read.go server/internal/store/queries/`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: 002, 004, 008 recommended first (behavior/tests stable)
- **Category**: perf
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Roadmap P2 (`docs/business/04-escopo-e-roadmap.adoc`): enriched query for
tenant resources in one DB round-trip. Today `BuildResourceFields` issues
`GetDefinition` + `ListFields` + `ListResourceValues` **per resource**.
`ResolveTenant` and admin `listTenantResources` loop that → `1+3R` queries on
the consumer hot path after ETag miss.

## Current state

```go
// resource_fields.go BuildResourceFields — 3 queries per resource
// resolve.go ResolveTenant — for _, res := range resources { resolveResource → BuildResourceFields }
// admin_read.go listTenantResources — same loop
```

Queries live in `server/internal/store/queries/resources.sql` (+ definitions).

**Architecture** (`docs/developers/01-arquitetura.adoc`): assembly stays in
`service`; sqlc for SQL; httpapi stays thin.

**RN-06**: admin masks secrets unless `reveal`; consumer always reveal=true.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| sqlc | `make sqlc` | exit 0 |
| Tests | `cd server && go test ./internal/service/ ./internal/httpapi/ -count=1` | exit 0 |
| Full | `cd server && go vet ./... && go test ./...` | exit 0 |

## Scope

**In**:
- New sqlc query(ies) joining resources → definitions → fields → values for a
  tenant (or resource id list)
- `service` assembly API used by resolve + admin list
- Tests for mask/reveal and resolve golden path

**Out**:
- Pagination of admin lists (separate)
- Caching API keys (separate)
- Frontend changes

## Git workflow

- Branch: `advisor/012-batch-resolve-resource-fields`
- Commit: `perf(resolve): batch load resource fields and values`
- No push/PR unless asked.

## Steps

### Step 1: Design query shape

Example approach (adjust to sqlc ergonomics):

```sql
-- name: ListResourceFieldValuesByTenant :many
SELECT
  tr.id AS tenant_resource_id,
  tr.status AS resource_status,
  rd.id AS definition_id,
  rd.key AS definition_key,
  rd.name AS definition_name,
  rf.id AS field_id,
  rf.key AS field_key,
  rf.label AS field_label,
  rf.data_type,
  rf.required,
  rf.is_secret,
  rf.sort_order,
  trv.value_plain,
  trv.value_cipher,
  trv.nonce,
  trv.key_version
FROM tenant_resources tr
JOIN resource_definitions rd ON rd.id = tr.resource_definition_id
JOIN resource_fields rf ON rf.resource_definition_id = rd.id
LEFT JOIN tenant_resource_values trv
  ON trv.tenant_resource_id = tr.id AND trv.resource_field_id = rf.id
WHERE tr.tenant_id = $1
  AND ($2::bool OR tr.status = 'active')  -- or two queries: active-only vs all
ORDER BY rd.key, rf.sort_order, rf.key;
```

Admin list needs all statuses; resolve needs active only. Either two queries
or a status filter param.

**Verify**: `make sqlc`

### Step 2: Service batch builder

Add `BuildTenantResources(ctx, deps, tenantID, reveal, activeOnly) ([]..., error)`
that:

1. Runs one (or two) queries
2. Groups rows by `tenant_resource_id`
3. Applies `presentValue` / encrypt decode per field
4. Returns the same shapes used today by admin JSON and resolve

Keep `BuildResourceFields` for single-resource paths if still needed, or
implement single as filter of batch.

**Verify**: unit tests with fake data or integration.

### Step 3: Wire resolve + admin

- `ResolveTenant` / `resolveResource` use batch when resolving many
- `listTenantResources` uses batch

**Verify**: `TestE2E_AdminCreateThenResolve`, mask/reveal tests, ETag tests pass.

### Step 4: Confirm query count (manual or test)

Optional: wrap pool with query counter in test — not required if too heavy.
Document expected O(1) queries per tenant resolve in PR notes.

## Done criteria

- [x] Resolve path does not call GetDefinition/ListFields/ListResourceValues
      once per resource in a loop (grep the hot path)
- [x] Admin mask/reveal behavior unchanged
- [x] Consumer decrypt behavior unchanged
- [x] Full server tests pass
- [x] README `DONE`

## STOP conditions

- sqlc cannot express the join cleanly — STOP with alternative design (two
  queries: list resources+defs, then `WHERE tenant_resource_id = ANY($1)`)
- Behavior drift on empty fields / missing values — match old builder semantics
  exactly before optimizing further

## Maintenance notes

- Roadmap item P2 — mark done in docs when shipped.
- Reviewer: secret mask paths and ordering of fields.
