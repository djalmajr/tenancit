# Plan 007: Normalize hostnames on write and resolve (case-insensitive DNS)

> **Executor instructions**: Follow step by step; verify each step; STOP on
> conditions below. Update `plans/README.md` when done (unless reviewer owns index).
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/admin.go server/internal/httpapi/consumer.go server/internal/service/resolve.go server/internal/httpapi/integration_test.go`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

DNS hostnames are case-insensitive. The service stores and matches hostnames
exactly as typed. Registering `App.Example.com` makes `app.example.com` resolve
404, and case variants can both exist as distinct unique rows. Edge injectors
typically send lowercase host headers — tenants “disappear” depending on how
the domain was typed in the admin UI.

## Current state

- `admin.go` `addDomain`: stores `in.Hostname` raw.
- `tenants.sql` `GetTenantByHostname`: `WHERE d.hostname = $1` exact match.
- `consumer.go` passes query/path hostname straight to resolver.
- Unique constraint on `tenant_domains.hostname` is case-sensitive text.

**Product note**: docs say “exact hostname match” (RN-02) meaning no wildcards,
not “case-sensitive DNS”. Canonicalize to lowercase ASCII hostnames.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Tests | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |
| Full | `cd server && go vet ./... && go test ./...` | exit 0 |

## Scope

**In**:
- Shared normalize helper (e.g. `service/hostname.go` or `httpapi/hostname.go`)
- `addDomain` write path
- `handleIdentify`, `handleResolve`, `handleResolveOne` (and any other hostname input)
- Integration tests

**Out**:
- Punycode/IDN full unicode normalization beyond simple `strings.ToLower` + trim
  (if non-ASCII hosts appear, STOP and report — do not invent IDNA stack)
- Bulk migration job for existing mixed-case rows (document one-shot SQL in plan
  maintenance notes if needed)
- Changing unique constraint to `citext` (optional follow-up)

## Git workflow

- Branch: `advisor/007-normalize-hostnames`
- Commit: `fix: canonicalize hostnames to lowercase for match`
- No push/PR unless asked.

## Steps

### Step 1: Helper

```go
// CanonicalHostname lowercases and trims spaces and a single trailing dot.
func CanonicalHostname(h string) string {
	h = strings.TrimSpace(h)
	h = strings.TrimSuffix(h, ".")
	return strings.ToLower(h)
}
```

Reject empty after normalize with existing 400 paths.

**Verify**: `cd server && go build ./...`

### Step 2: Apply on write and read

- `addDomain`: store `CanonicalHostname(in.Hostname)`; if empty → 400.
- Consumer: canonicalize query `hostname` and path `{hostname}` before lookup.

**Verify**: build green.

### Step 3: Tests

1. Add domain as `App.Example.COM` → stored/list shows lowercase
   `app.example.com` (or at least resolve works for lowercase).
2. Resolve/identify with mixed case → 200 same tenant as lowercase.
3. Second domain with different case of same host → 409 conflict.

**Verify**: `cd server && go test ./internal/httpapi/ -count=1` → exit 0

## Done criteria

- [x] Write + resolve paths use the same canonical form
- [x] Case-variant resolve tests pass
- [x] No out-of-scope files
- [x] README status `DONE`

## STOP conditions

- Existing production data has intentional mixed-case distinct hosts that must
  remain distinct (unlikely for DNS) — STOP.
- IDN/punycode required for a failing real hostname in tests — STOP and report.

## Maintenance notes

- One-shot ops: `UPDATE tenant_domains SET hostname = lower(hostname);` if
  duplicates appear, resolve manually before unique index conflicts.
- Reviewer: ensure resolve-one path param is normalized.
