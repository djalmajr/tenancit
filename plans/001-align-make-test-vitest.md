# Plan 001: Align `make test` with CI by running Vitest

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- Makefile docs/developers/06-operacao-e-testes.adoc web/package.json .github/workflows/ci.yml`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests | dx
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

CI runs `bun run test` (Vitest) for the SPA, but the local default gate
`make test` only typechecks the web package. Agents and humans can ship SPA
regressions that CI later catches, or get a false green signal offline.
Aligning Makefile with CI is a small change that makes every later plan’s
verification trustworthy.

## Current state

- `Makefile` — build/test targets for the monorepo.
- `web/package.json` — defines `"test": "vitest run"`.
- `.github/workflows/ci.yml` — web job runs `bun run test` then `bun run build`.
- `docs/developers/06-operacao-e-testes.adoc` — documents gates; currently says
  `make test-web` is only `tsc`.

Excerpt `Makefile` (as of plan write):

```makefile
## test: unit tests (Go + web typecheck). Use test-db for DB-backed tests.
test: test-go test-web

test-go:
	cd server && go vet ./... && go test ./...

test-web:
	cd web && bunx tsc --noEmit
```

Excerpt `docs/developers/06-operacao-e-testes.adoc`:

```
`make test-web` roda `bunx tsc --noEmit`.

Para rodar a suíte web com Vitest:

cd web && bun run test
```

**Conventions**: Makefile uses tabs for recipe lines. Prefer updating the
documented gate so `make test` is the one-command story.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Web unit | `cd web && bun run test` | exit 0 |
| Web typecheck | `cd web && bunx tsc --noEmit` | exit 0 |
| Full gate | `make test` | exit 0 (Go needs Docker for non-skipped integration tests) |

## Scope

**In scope**:
- `Makefile`
- `docs/developers/06-operacao-e-testes.adoc` (gate section only)

**Out of scope**:
- Adding new Vitest cases (other plans)
- Changing CI workflow (already correct)
- `make test-db` / testcontainers behavior
- Root `package.json` / npm vs Bun cleanup

## Git workflow

- Branch: `advisor/001-align-make-test-vitest` (or work on the branch your
  dispatcher gave you)
- Commit style (from recent history): conventional commits, e.g.
  `chore: run vitest in make test-web`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Update `test-web` to typecheck + Vitest

Change `Makefile` so `test-web` runs both typecheck and unit tests, matching CI
order (typecheck then test is fine; CI does typecheck then test then build):

```makefile
test-web:
	cd web && bunx tsc --noEmit && bun run test
```

Optionally update the comment above `test:` to say “Go + web typecheck/unit”
instead of only typecheck.

**Verify**: `make test-web` → exit 0; output includes Vitest run and no failing tests.

### Step 2: Update developer docs

In `docs/developers/06-operacao-e-testes.adoc`, change the “Gates locais”
section so:

- `make test-web` is described as typecheck **and** Vitest.
- Remove or shorten the separate “Para rodar a suíte web com Vitest” block that
  implies Vitest is only a manual extra step (or note it is already in
  `make test-web`).

**Verify**: `rg -n "test-web|Vitest|vitest" docs/developers/06-operacao-e-testes.adoc`
→ docs no longer claim `test-web` is typecheck-only.

### Step 3: Full gate smoke

**Verify**: `make test` → exit 0.

If Docker is unavailable, Go integration tests may `t.Skip`; that is pre-existing.
Vitest and `tsc` must still run and pass.

## Test plan

- No new test files.
- Confirmation is that existing Vitest suite is invoked by `make test-web`.

## Done criteria

- [x] `make test-web` runs `tsc` and Vitest (visible in output)
- [x] `make test-web` exits 0
- [x] `docs/developers/06-operacao-e-testes.adoc` matches the Makefile behavior
- [x] No files outside the in-scope list are modified
- [x] `plans/README.md` status for 001 → `DONE`

## STOP conditions

- Existing Vitest suite fails before any Makefile change — report failures; do
  not “fix” product code under this plan unless a one-line flaky config is
  clearly the cause.
- `bun` is missing — report; do not switch the project to npm.
- CI already diverges from this plan’s intent in unexpected ways after drift.

## Maintenance notes

- Future web test additions must pass under `make test` / CI automatically.
- Reviewer: ensure `test-web` still fails the make target when Vitest fails
  (no `|| true`).
