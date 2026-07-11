# Plan 013: Root README and `.env.example` for onboarding

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- README.md .env.example docs/developers/05-ambiente-local.adoc Makefile docker-compose.yml`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx | docs
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

There is no root `README.md`. Setup lives only under `docs/developers/`.
Agents and new contributors miss one-command paths (`make dev-compose`,
`make test`, required env vars). `.env.example` is missing for bare-metal runs.

## Current state

- Docs: `docs/developers/05-ambiente-local.adoc`, `06-operacao-e-testes.adoc`
- Compose demo env in `docker-compose.yml` (demo token **type** only — never
  copy production secrets into docs)
- `Makefile` targets: `dev-compose`, `build`, `test`, `docker-up`

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Doc only | files exist | `test -f README.md && test -f .env.example` |

## Scope

**In**:
- `README.md` (root)
- `.env.example` (names + placeholders only — **no real secrets**)
- Optional one-line pointer in `docs/README.adoc`

**Out**:
- Rewriting full AsciiDoc manual
- Committing `.env` with values
- Changing runtime defaults

## Git workflow

- Branch: `advisor/013-root-readme-env-example`
- Commit: `docs: add root README and .env.example`
- No push/PR unless asked.

## Steps

### Step 1: README.md

Include:

1. What Tenancit is (1 short paragraph — control plane for multi-tenant
   resource config)
2. Requirements: Go 1.25, Bun 1.3, Docker
3. Quick start:
   - `docker compose up --build` → http://localhost:8080
   - Dev: `make dev-compose` → Vite :5180, API :8081
4. Env vars table (names only): `TENANCIT_DATABASE_URL`, `TENANCIT_ADMIN_TOKEN`,
   `TENANCIT_AES_KEY`, `TENANCIT_AES_KEY_VERSION`, `TENANCIT_ADDR`
5. Common make targets: `make test`, `make build`, `make sqlc`
6. Link to `docs/README.adoc` and ADR folder
7. Note: demo compose token is for local only; rotate in real deploys

**Never** paste live production credentials. For local demo you may mention
that compose sets a **demo** admin token (name the env var; if you cite the
dev default, label it clearly as non-production). Prefer pointing to compose
file rather than repeating secrets in multiple places.

### Step 2: `.env.example`

```bash
TENANCIT_ADDR=:8080
TENANCIT_DATABASE_URL=postgres://postgres:tenancit@localhost:5432/tenancit?sslmode=disable
TENANCIT_ADMIN_TOKEN=change-me
TENANCIT_AES_KEY=  # base64 of 32 bytes — generate for real deploys
TENANCIT_AES_KEY_VERSION=1
```

### Step 3: Ensure `.gitignore` ignores `.env` if not already

**Verify**: `rg '^\.env$' .gitignore` or equivalent — `.env` should not be
committed with secrets.

## Done criteria

- [x] Root README exists with runnable quick start
- [x] `.env.example` lists required vars without real secrets
- [x] README `DONE`

## STOP conditions

- Repo intentionally has no root README for a documented monorepo reason —
  unlikely; proceed.

## Maintenance notes

- Keep README short; deep content stays in `/docs`.
