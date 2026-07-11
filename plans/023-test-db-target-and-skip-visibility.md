# Plan 023: Fix `make test-db` story and Docker skip visibility

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- Makefile server/internal/testsupport/ docs/developers/06-operacao-e-testes.adoc`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: 001 optional
- **Category**: tests | dx
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

1. `make test-db` spins Postgres on 55432 and sets `TEST_DATABASE_URL`, but
   almost all tests use testcontainers via `testsupport.NewDB` and ignore that
   env (only migrate_test reads it).
2. When Docker is down, `NewDB` **skips** — `make test` exits 0 while skipping
   the critical path. Local false green.

## Current state

- `Makefile` `test-db`, `pg-test-up/down`
- `testsupport/postgres.go` — Skip on docker failure
- Docs mention both patterns

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Unit path | `cd server && go test ./internal/crypto/ ./internal/service/ -count=1` | exit 0 offline |
| With Docker | `cd server && go test ./...` | exit 0, no unexpected skip of httpapi if Docker up |

## Scope

**In**: Makefile, testsupport, migrate_test wiring, docs 06

**Out**: Replacing testcontainers entirely; CI docker-in-docker changes beyond docs

## Git workflow

- Branch: `advisor/023-test-db-target-and-skip-visibility`
- Commit: `chore(test): clarify DB test gates and skip policy`
- No push/PR unless asked.

## Steps

### Step 1: Choose one story (recommended)

**Canonical**: testcontainers via `testsupport.NewDB` for all DB tests.
Migrate tests should use `NewDB` too; deprecate `TEST_DATABASE_URL` path.

- Update `migrate_test.go` to use testsupport
- Change `make test-db` to either:
  - alias of `go test` with a note that Docker is required, or
  - remove `pg-test-*` if unused

### Step 2: Skip policy

In `testsupport.NewDB`:

- Keep Skip by default for offline unit workflows
- If `REQUIRE_DB_TESTS=1` (set in CI docs / optional Makefile), **Fatal**
  instead of Skip when Docker unavailable

Document in `06-operacao-e-testes.adoc`.

### Step 3: Makefile

```makefile
test-go:
	cd server && go vet ./... && go test ./...

# Optional strict:
# test-go-strict:
#	cd server && REQUIRE_DB_TESTS=1 go test ./...
```

**Verify**: with Docker, tests pass; without Docker, crypto/service still pass;
with REQUIRE_DB_TESTS=1 and no Docker, failure is loud.

## Done criteria

- [x] One documented DB test story
- [x] Dead pg-test path removed or clearly secondary
- [x] Optional strict mode documented
- [x] README `DONE`

## STOP conditions

- CI cannot run Docker — do not set REQUIRE_DB_TESTS=1 in CI without Docker.

## Maintenance notes

- CI already has Docker on ubuntu-latest for testcontainers.
