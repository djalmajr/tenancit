# Plan 005: Omit `key_hash` from admin API client JSON

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in STOP conditions occurs, stop and report.
> When done, update `plans/README.md` unless the reviewer maintains the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- server/internal/httpapi/admin.go server/internal/store/db/models.go web/src/lib/api.ts web/src/routes/api-clients.tsx`
> On mismatch with "Current state", STOP.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

API clients store only SHA-256 of the bearer token (`key_hash`). Admin
`POST/GET /v1/admin/api-clients` currently JSON-encodes the raw sqlc
`ApiClient` model, which includes `key_hash`. The SPA never displays it
(it expects optional `key_preview` and shows a placeholder). The hash still
lands in browser memory, DevTools, HAR files, and proxies — unnecessary
credential material beyond “token shown once at creation” (RN-09 / ADR 0004).

## Current state

Generated model (`server/internal/store/db/models.go`):

```go
type ApiClient struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	KeyHash   string    `json:"key_hash"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
```

Handlers (`server/internal/httpapi/admin.go`):

```go
// create — returns map with full client + raw token once
writeJSON(w, http.StatusCreated, map[string]any{"client": c, "token": token})

// list
cs, err := s.Q.ListAPIClients(...)
writeJSON(w, http.StatusOK, cs)
```

SPA (`web/src/lib/api.ts`):

```ts
export interface ApiClient {
  id: string;
  name: string;
  key_preview?: string;
  status: string;
  created_at?: string;
}
```

UI shows `client.key_preview ?? "tnc_••••••••"` — server never sets
`key_preview` today.

**Do not edit sqlc models by hand** — they are generated. Map to a DTO in
httpapi instead.

**ADR 0004**: raw API client token appears once at creation; only hash in DB.

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| Go tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |
| Web typecheck | `cd web && bunx tsc --noEmit` | exit 0 |
| Grep | `rg 'key_hash' server/internal/httpapi web/src` | no admin JSON emission of hash |

## Scope

**In scope**:
- `server/internal/httpapi/admin.go` (create/list API clients; status handler
  if it returns full client)
- Possibly small DTO helper in `server/internal/httpapi/`
- `server/internal/httpapi/admin_actions.go` if `setAPIClientStatus` returns
  full `db.ApiClient`
- Tests in `server/internal/httpapi/*_test.go`
- Optional SPA touch only if types need `createdAt` camelCase — prefer
  **matching existing SPA field names** (`created_at` already expected)

**Out of scope**:
- Regenerating sqlc just to hide JSON tags (wrong layer)
- Changing token generation / hash algorithm
- Implementing a real `key_preview` from the raw token at create time is
  **optional** (nice-to-have); if you add it, only on create response, derived
  from the raw token (e.g. first 8 chars of `tnc_…`), **never** from the hash
- Auth middleware changes

## Git workflow

- Branch: `advisor/005-omit-api-client-key-hash`
- Commit: `fix(admin): omit api client key_hash from JSON responses`
- Do NOT push/PR unless asked.

## Steps

### Step 1: Define a public DTO

In `httpapi` (e.g. top of `admin.go` or `api_client_dto.go`):

```go
type apiClientView struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	// optional:
	// KeyPreview string `json:"key_preview,omitempty"`
}

func viewAPIClient(c db.ApiClient) apiClientView {
	return apiClientView{
		ID: c.ID, Name: c.Name, Status: c.Status, CreatedAt: c.CreatedAt,
	}
}
```

**Never** copy `c.KeyHash` into any JSON struct tag that is returned.

### Step 2: Wire create + list (+ status)

**createAPIClient**:

```go
writeJSON(w, http.StatusCreated, map[string]any{
  "client": viewAPIClient(c),
  "token":  token, // still shown once — required by product
})
```

**listAPIClients**:

```go
out := make([]apiClientView, 0, len(cs))
for _, c := range cs {
  out = append(out, viewAPIClient(c))
}
writeJSON(w, http.StatusOK, out)
```

**setAPIClientStatus** (in `admin_actions.go`): return `viewAPIClient(c)` not raw `c`.

**Verify**: `cd server && go build ./...` → exit 0

### Step 3: Test

Add a focused test (integration or admin_test) that:

1. Creates an API client via admin.
2. Unmarshals create response into `map[string]any` / nested client.
3. Asserts `client` has `id`, `name`, `status` and **does not** contain key
   `key_hash` (check map keys).
4. Lists clients; same assertion for each element.
5. Asserts create response still includes top-level `token` non-empty starting
   with `tnc_`.

Do **not** print or hardcode real production secrets; test token is ephemeral.

**Verify**: `cd server && go test ./internal/httpapi/ -count=1 -run APIClient`
(or your test name) → pass; full `go test ./internal/httpapi/` → pass.

### Step 4: SPA sanity

`cd web && bunx tsc --noEmit` → exit 0

No SPA change required if JSON field names stay `id/name/status/created_at`.

## Test plan

| Case | Expected |
|------|----------|
| Create client | 201; `token` present; `client` without `key_hash` |
| List clients | 200; no `key_hash` fields |
| Status toggle | 200; no `key_hash` |
| Auth still works with returned token | resolve/identify 200 (optional) |

## Done criteria

- [x] Admin API client responses never include `key_hash`
- [x] Create still returns one-time `token`
- [x] Automated test asserts absence of `key_hash`
- [x] `go test ./internal/httpapi/` and web `tsc` exit 0
- [x] Scope respected
- [x] `plans/README.md` 005 → `DONE`

## STOP conditions

- You find another endpoint that must return the hash for a documented reason —
  none known; STOP and report rather than re-adding hash.
- Editing `models.go` by hand seems “easier” — **do not**; use DTO.

## Maintenance notes

- Reviewer: network tab / test must prove no `key_hash`.
- Optional follow-up: set `key_preview` on create only (`tnc_` + first few
  hex chars of the raw token) so the table is slightly more useful.
- sqlc model can keep `KeyHash` for DB scanning forever.
