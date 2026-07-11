# Plan 010: Cap JSON body size and complete HTTP server timeouts

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/cmd/server/main.go server/internal/httpapi/admin.go server/internal/httpapi/`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (004 helpers optional)
- **Category**: security
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Admin `decode` uses `json.NewDecoder(r.Body)` with no size limit. Authenticated
admin can POST multi-GB JSON and pressure memory. `http.Server` only sets
`ReadHeaderTimeout` (5s) — no `ReadTimeout` / `WriteTimeout` / `IdleTimeout`.

## Current state

```go
// admin.go
func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

// main.go
httpServer := &http.Server{
  Addr: addr,
  Handler: srv.Routes(staticHandler),
  ReadHeaderTimeout: 5 * time.Second,
}
```

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Build | `cd server && go build ./...` | exit 0 |
| Tests | `cd server && go test ./...` | exit 0 |

## Scope

**In**:
- `decode` (or middleware) with `http.MaxBytesReader`
- `cmd/server/main.go` timeouts
- Optional unit test for oversized body → 413/400

**Out**:
- Rate limiting / WAF
- TLS configuration
- Changing SPA upload features (none today)

## Git workflow

- Branch: `advisor/010-request-limits-timeouts`
- Commit: `fix(http): limit JSON body size and set server timeouts`
- No push/PR unless asked.

## Steps

### Step 1: MaxBytesReader in decode

```go
const maxJSONBody = 1 << 20 // 1 MiB

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // optional; if it breaks clients, omit
	return dec.Decode(v)
}
```

Note: `MaxBytesReader` needs `http.ResponseWriter` to auto-413 in some
patterns. Prefer:

```go
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	return json.NewDecoder(r.Body).Decode(v)
}
```

Update all `decode(r, &in)` call sites to `decode(w, r, &in)` if signature
changes. On `*http.MaxBytesError`, return 413 with stable
`{"error":"request body too large"}` — not `err.Error()`.

**Do not** enable `DisallowUnknownFields` if existing clients send extra
fields — check tests first; default is leave decoder permissive.

**Verify**: `go build ./...`; existing admin tests pass.

### Step 2: Server timeouts

```go
httpServer := &http.Server{
	Addr:              addr,
	Handler:           srv.Routes(staticHandler),
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       15 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```

Avoid WriteTimeout so short it breaks large SPA asset responses on slow
links — 30s is a starting point for this app.

**Verify**: `go build -o /dev/null ./cmd/server`

### Step 3: Test oversized body (optional but preferred)

Admin POST with body > 1MiB → 413 or 400; message stable.

**Verify**: `go test ./internal/httpapi/ -count=1`

## Done criteria

- [x] JSON decode limited (~1MiB)
- [x] Server has Read/Write/Idle timeouts
- [x] Tests still pass
- [x] README `DONE`

## STOP conditions

- Legitimate payloads need >1MiB (none known) — raise limit with justification.
- Timeout values break existing long testcontainers tests in-process — those
  use `httptest`, not real server; should be fine.

## Maintenance notes

- Reviewer: all `decode` call sites updated.
- Document limit in `docs/developers/03-contratos-http.adoc` if public.
