# Plan 017: CI full product build + crypto.FromEnv tests

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- .github/workflows/ci.yml server/internal/crypto/ Makefile`

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests | dx
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

CI runs server tests and web build separately but never `make build` / embed /
Go binary with SPA. Embed path regressions only show up locally or on image
build. `crypto.FromEnv` is untested while `main` depends on it for boot.

## Current state

- `.github/workflows/ci.yml` — jobs `server` and `web`
- `Makefile` `build`: build-web → embed → build-server
- `crypto/config.go` `FromEnv` loads `TENANCIT_AES_KEY`, version, optional
  `TENANCIT_AES_KEY_V<n>`

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Local build | `make build` | exit 0; `server/bin/server` exists |
| Crypto tests | `cd server && go test ./internal/crypto/ -count=1` | exit 0 |

## Scope

**In**:
- `.github/workflows/ci.yml` (new job or steps)
- `server/internal/crypto/*_test.go` for FromEnv
- Optional Makefile target already exists — wire CI to it

**Out**: Publishing images to a registry; multi-arch

## Git workflow

- Branch: `advisor/017-ci-full-build-and-crypto-env-tests`
- Commit: `ci: full make build; test crypto FromEnv`
- No push/PR unless asked.

## Steps

### Step 1: crypto.FromEnv tests

Using `t.Setenv`:

1. Missing key → error
2. Invalid base64 → error
3. Valid 32-byte key base64 + version → Encrypt/Decrypt round-trip
4. Multi-version: current v2 + V1 present → decrypt ciphertext sealed with v1

**Never** commit real production keys; generate ephemeral in test.

**Verify**: `go test ./internal/crypto/ -count=1`

### Step 2: CI job `product-build`

After or parallel to web/server:

```yaml
  product:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.25", cache-dependency-path: server/go.sum }
      - uses: oven-sh/setup-bun@v2
        with: { bun-version: "1.3" }
      - name: make build
        run: make build
      - name: binary exists
        run: test -x server/bin/server
```

**Verify**: workflow YAML valid; local `make build` works.

## Done criteria

- [x] FromEnv cases tested
- [x] CI exercises embed+Go binary path
- [x] README `DONE`

## STOP conditions

- CI minutes budget forbids full build — use `make embed` after web artifact
  upload instead; do not skip entirely without reporting.

## Maintenance notes

- Cache Bun/Go in the product job for speed.
