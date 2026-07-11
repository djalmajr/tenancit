# Plan 024: HTTP response hardening headers for SPA + API

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- server/internal/httpapi/server.go server/internal/spa/`

## Status

- **State**: DONE (2026-07-09)
- **Priority**: P3
- **Effort**: S–M
- **Risk**: MED (strict CSP can break SPA if misconfigured)
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Admin SPA holds bootstrap token in `localStorage` and can load cleartext
secrets. No CSP, `X-Content-Type-Options`, `Referrer-Policy`, or
`frame-ancestors` defenses. Reduces impact of future XSS.

## Current state

- `server.go` Routes: RequestID + Recoverer only
- `spa` embeds hashed Vite assets under `/assets/`

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Build | `make build` or web build + go test | exit 0 |
| Manual | curl -I headers on `/` and `/healthz` | headers present |

## Scope

**In**: middleware in httpapi or spa handler headers

**Out**: Full XSS audit; nonce-based CSP with inline scripts (Vite build
should avoid inline)

## Git workflow

- Branch: `advisor/024-response-hardening-headers`
- Commit: `fix(http): security headers for SPA and API`
- No push/PR unless asked.

## Steps

### Step 1: Middleware

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("Referrer-Policy", "no-referrer")
w.Header().Set("X-Frame-Options", "DENY")
// CSP — start conservative for same-origin SPA:
// default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' (if needed for Tailwind);
// img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; object-src 'none'
```

Tune CSP until embedded SPA loads (fonts may need `font-src 'self'`).

### Step 2: Verify SPA

`make build` + run binary against local DB; load admin UI; no console CSP
blocks for assets.

### Step 3: Test

httptest check that `/healthz` includes `nosniff` (and CSP if applied globally).

## Done criteria

- [x] Hardening headers present
- [x] SPA still loads
- [x] README `DONE`

## Completion evidence

- `health_test.go` verifies the global security-header middleware.
- The packaged image built successfully and served the embedded SPA on `:8080`.
- `curl -I http://localhost:8080/` returned CSP, HSTS, `nosniff`,
  `DENY`, `no-referrer`, and the restrictive permissions policy.
- The CSP required neither `unsafe-eval` nor external asset origins.

## STOP conditions

- CSP requires `'unsafe-eval'` to work — STOP and report (should not for Vite
  prod build).

## Maintenance notes

- Re-test CSP after major frontend toolchain upgrades.
