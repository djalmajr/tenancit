# Thermo-Nuclear Code Quality Review

**Scope:** Initial commit on `main` (Konvario). Uncommitted changes are only `CLAUDE.md` / tooling — not reviewed here.

**Verdict: Do not approve.** Behavior largely matches the epic checklist, but several structural issues would compound quickly. The biggest gaps are non-atomic writes, duplicated read assembly, N+1 query patterns, and a reveal/auth contract that the UI effectively bypasses.

---

## 1. Structural regressions (presumptive blockers)

### 1.1 `createResource` is non-atomic — partial state on failure

```191:215:server/internal/httpapi/admin.go
	res, err := s.Q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: tenantID, ResourceDefinitionID: def.ID,
	})
	// ...
	for _, f := range fields {
		// ...
		if _, err := s.Q.UpsertResourceValue(ctx, p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
```

The resource row is committed before values are written. A mid-loop failure leaves an orphan `tenant_resources` row (possibly active, possibly empty). `WithTx` exists in sqlc but is unused.

**Code-judo move:** Extract `service.ProvisionResource(ctx, tx, tenantID, definitionKey, values)` that validates, encodes, inserts resource + all values in **one transaction**. The handler becomes decode → call service → write JSON. That deletes orchestration from the HTTP layer and removes the partial-state category entirely.

---

### 1.2 Admin API has no auth boundary; reveal is effectively public

Consumer routes use `RequireAPIKey`. Admin routes under `/v1/admin/*` have **zero** middleware.

Per `decisions.md`: *"`?reveal=true` com escopo de auth; default mascarado."* The server implements masking, but without admin auth, anyone who reaches the port can CRUD tenants, create API clients, and call `?reveal=true`.

This is feature logic (secret reveal) living on a shared, unauthenticated path. Even with "minimal panel auth" as out-of-scope, the reveal contract needs a boundary now — otherwise RN-06 is security theater.

**Remedy:** At minimum, gate admin (and especially `reveal=true`) behind the same API-key middleware or a dedicated admin token. Do not ship reveal as a query-param toggle on an open endpoint.

---

### 1.3 RN-06 is implemented twice, inconsistently, and the UI bypasses the server contract

Server masking (good):

```50:56:server/internal/service/values.go
func presentValue(c *crypto.Cryptor, isSecret bool, v db.TenantResourceValue, reveal bool) (string, error) {
	if isSecret && !reveal {
		return MaskedValue, nil
	}
	return decodeValue(c, v)
}
```

UI always requests cleartext:

```48:52:web/src/routes/tenant-detail.tsx
    const [t, d, r] = await Promise.all([
      api.getTenant(id).catch(() => null),
      api.listDomains(id).catch(() => []),
      api.listTenantResources(id, true).catch(() => []),
```

Then `RevealValue` masks again client-side with a **different** placeholder (`••••••••••••` vs server `••••••••`).

Three layers, two mask characters, secrets always on the wire. Pick one model:

| Approach | Behavior |
|---|---|
| **A (server-owned)** | Default `reveal=false`; reveal only on explicit user action (`?reveal=true` or per-field endpoint) |
| **B (client-owned)** | Always `reveal=true` for trusted admin session; delete server masking for admin reads |

Current hybrid adds complexity without buying safety.

---

## 2. Missed code-judo moves (high conviction)

### 2.1 Duplicated resource assembly in two read paths

`listTenantResources` (`admin_read.go`) and `resolveResource` (`resolve.go`) share the same shape:

1. Get definition
2. List fields
3. List values
4. Join by `resource_field_id`
5. Decode / present

**Code-judo move:** One canonical assembler, e.g. `service.BuildFieldValues(ctx, q, cryptor, resourceID, mode Admin|Consumer, reveal bool)`. Admin and consumer handlers become thin wrappers. That deletes a whole category of drift (and half the N+1 surface).

---

### 2.2 N+1 queries baked into hot read paths

| Endpoint | Pattern |
|---|---|
| `listDefinitions` | 1 + N `ListFields` per definition; errors silently dropped (`fields, _ := ...`) |
| `listTenantResources` | Per resource: `GetDefinition` + `ListFields` + `ListResourceValues` |
| `overview` | Per tenant: `ListTenantDomains` + `ListActiveResourcesByTenant`; partial errors ignored |
| `ByHostname` | Per resource: 3 queries in `resolveResource` |
| `ByHostnameAndDefinition` | Resolves **all** resources, then scans for one match |

**Code-judo move:** Push aggregation into SQL (JOINs, `COUNT(*) FILTER`, or a single `ListTenantResourcesEnriched` query). `overview` especially should be one query — it is literally an aggregation dashboard doing O(tenants) round-trips.

`ByHostnameAndDefinition` should not call `ByHostname` + linear search. Add `GetActiveResourceByTenantAndDefinitionKey` or resolve one resource directly.

---

### 2.3 Business logic lives in the HTTP handler, not the service layer

`createResource` does definition lookup, status check (RN-08), required-field validation (RN-03), encryption (RN-04), and multi-step persistence — all in `admin.go`. Reads went through `Resolver`; writes did not get the same treatment.

Writes and reads should be symmetric: **handlers parse/validate HTTP; service owns invariants and transactions.**

---

### 2.4 `tenant-detail.tsx` is becoming a god-component (411 lines)

Three dialogs, tab state, resource provisioning wizard, domain CRUD, edit form, shared `err` state across dialogs. Not past 1k yet, but the trajectory is wrong.

**Decompose before it crosses ~500:**

- `TenantResourcesTab` / `TenantDomainsTab`
- `useTenantDetail(id)` for parallel loading (already partially there)
- Per-dialog error state (shared `err` can leak errors across modals)
- Shared `PageHeader`, `StatusBadge`, `CrudDialog` primitives across routes (tenants, definitions, api-clients duplicate the same load/dialog/badge patterns)

---

## 3. Spaghetti / branching growth

### 3.1 Copy-pasted status validation

```62:64:server/internal/httpapi/admin_actions.go
	if err := decode(r, &in); err != nil || (in.Status != "active" && in.Status != "inactive") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
```

Same pattern three times with slightly different allowed sets (`active|inactive` vs `active|revoked`). A typed `Status` enum or `parseStatus(in, allowed...)` would collapse three branches into one concept.

### 3.2 Duplicate ID parsers

`parseID` (`admin.go`) and `parseParam` (`admin_actions.go`) do the same thing with different param names. One helper: `parseUUIDParam(r, name string)`.

### 3.3 Silent error swallowing on the frontend

`.catch(() => {})` appears on load, delete, toggle, and status changes across every route. Failures are invisible — users see stale UI with no feedback.

`toggleResource` is especially bad:

```111:117:web/src/routes/tenant-detail.tsx
  async function toggleResource(r: TenantResource) {
    try {
      await api.setResourceStatus(id, r.id, r.status === "active" ? "inactive" : "active");
      await load();
    } catch {
      /* RN-01 conflict on reactivate — ignore silently */
    }
  }
```

RN-01 conflicts should surface a toast/message, not vanish.

---

## 4. Boundary / abstraction / type-contract problems

### 4.1 Thin wrappers that add indirection without clarity

`present.go` is a one-line passthrough:

```8:12:server/internal/service/present.go
func PresentValue(c *crypto.Cryptor, isSecret bool, v db.TenantResourceValue, reveal bool) (string, error) {
	return presentValue(c, isSecret, v, reveal)
}
```

Same for `EncodeValueFor` → `encodeValue`. Either export the real functions from `values.go` or keep them unexported and call from `httpapi` within the same package pattern — but a separate file for a passthrough buys nothing.

### 4.2 Mixed JSON casing across API boundaries

- sqlc models: `data_type`, `is_secret` (snake_case)
- Admin read DTOs: `dataType`, `isSecret` (camelCase)
- Frontend `Field` vs `ResourceFieldValue` use different conventions

This works today (Go's case-insensitive JSON matching), but it is a latent contract bug. One canonical JSON shape per surface, or explicit struct tags everywhere.

### 4.3 Magic status strings scattered

`"active"`, `"inactive"`, `"revoked"` appear in Go handlers, SQL partial indexes, and React components with no shared typed model. Every new status adds another special-case branch in multiple files.

### 4.4 `overview` silently degrades on partial failure

```179:203:server/internal/httpapi/admin_actions.go
		domains, _ := s.Q.ListTenantDomains(ctx, t.ID)
		// ...
		resources, _ := s.Q.ListActiveResourcesByTenant(ctx, t.ID)
		// ...
	defs, err := s.Q.ListDefinitions(ctx)
	if err == nil { ... }
	clients, err := s.Q.ListAPIClients(ctx)
	if err == nil { ... }
```

Dashboard numbers can be wrong without any error response. Aggregation endpoints should fail loud or return explicit partial flags — not quietly return zeros.

---

## 5. File-size / decomposition

No file crosses 1k lines. Largest: `tenant-detail.tsx` (411). Admin handlers are reasonably split (`admin.go` / `admin_read.go` / `admin_actions.go`). Decompose `tenant-detail` proactively — don't wait for 1k.

---

## 6. What is working well

- **Stack choice** (chi + pgx + sqlc + goose) is direct and legible — no magic ORM layer.
- **`Resolver`** is a clean read abstraction with a proper `ResolveQuerier` interface.
- **`crypto` package** is small, focused, version-aware.
- **Tests** are strong — mutation-aware comments (`RN-03`, `RN-01`, etc.) show intentional invariant guarding.
- **`RevealValue` extraction** for testability is the right instinct (though the reveal model above still needs resolution).
- **Parallel fetch in `tenant-detail.load`** (`Promise.all`) is correct orchestration.

---

## Prioritized action list

| Priority | Action |
|---|---|
| P0 | Wrap `createResource` in a transactional `service.ProvisionResource` |
| P0 | Add admin auth; tie `reveal=true` to that boundary |
| P0 | Resolve RN-06: server-owned reveal **or** client-owned — not both |
| P1 | Extract shared field-value assembly; kill duplicate read paths |
| P1 | Replace N+1 reads with SQL aggregation (especially `overview`, `listDefinitions`, `listTenantResources`) |
| P1 | Fix `ByHostnameAndDefinition` to resolve one resource directly |
| P2 | Decompose `tenant-detail.tsx`; extract shared route primitives |
| P2 | Replace silent `.catch(() => {})` with user-visible error handling |
| P2 | Collapse status validation + ID parsing duplication |
| P3 | Delete thin wrapper files; unify JSON casing contracts |

---

## Approval bar check

| Criterion | Status |
|---|---|
| No structural regression | **Fail** — non-atomic writes, open admin API |
| No obvious code-judo missed | **Fail** — transactional service + unified assembler are clear wins |
| No unjustified file-size explosion | **Pass** |
| No spaghetti from special-case branching | **Fail** — status strings, silent catches, dual reveal |
| No hacky abstractions | **Fail** — dual masking, thin wrappers |
| No boundary leaks | **Fail** — admin unauthenticated, write logic in handlers |
| No canonical-helper duplication | **Fail** — read assembly duplicated |

**Bottom line:** The skeleton is sound and tests show real discipline. Before calling this done, land the transactional write path, fix the admin/reveal security boundary, and consolidate the duplicated read assembly. Those three changes would make the rest of the cleanup (N+1, frontend decomposition) much more straightforward.