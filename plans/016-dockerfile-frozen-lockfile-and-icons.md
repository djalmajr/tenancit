# Plan 016: Strict Bun lockfile in Docker + drop unused Remix icon dep

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- Dockerfile web/package.json web/bun.lock web/src/components/ui/select.tsx`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dependencies | dx
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

1. `Dockerfile` runs `bun install --frozen-lockfile || bun install`, allowing
   unlocked installs when the lockfile fails — image can drift from CI.
2. Both `lucide-react` and `@remixicon/react` are dependencies; only
   `select.tsx` uses Remix (~3 icons). Extra bundle and dual icon systems.

## Current state

```dockerfile
RUN bun install --frozen-lockfile || bun install
```

`web/package.json` lists both icon packages; `select.tsx` imports Remix.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Install | `cd web && bun install` | exit 0 |
| Build web | `cd web && bun run build` | exit 0 |
| Docker (optional) | `docker build -t tenancit:plan016 .` | exit 0 |

## Scope

**In**: `Dockerfile`, `web/src/components/ui/select.tsx`, `web/package.json`,
`web/bun.lock`

**Out**: Root npm `package.json` cleanup (can note in maintenance); full font
subsetting (optional stretch)

## Git workflow

- Branch: `advisor/016-dockerfile-frozen-lockfile-and-icons`
- Commit: `chore: frozen bun lockfile in Docker; drop remixicon`
- No push/PR unless asked.

## Steps

### Step 1: Dockerfile

```dockerfile
RUN bun install --frozen-lockfile
```

Remove `|| bun install`.

**Verify**: `docker build` if Docker available; else ensure `web/bun.lock`
committed and `bun install --frozen-lockfile` works locally.

### Step 2: Replace Remix icons in select.tsx

Swap to `lucide-react` equivalents (`Check`, `ChevronDown`, `ChevronUp` or
whatever matches usage). Remove `@remixicon/react` from `package.json`.
Run `bun install` to refresh lockfile.

**Verify**: `cd web && bunx tsc --noEmit && bun run build`

## Done criteria

- [x] No fallback unlocked install in Dockerfile
- [x] `@remixicon/react` removed; select still works visually (build OK)
- [x] README `DONE`

## STOP conditions

- Frozen lockfile fails because lock is out of date — fix lockfile properly,
  do not restore `|| bun install`.

## Maintenance notes

- Optional follow-up: remove root npm-only `shadcn` tree if unused.
