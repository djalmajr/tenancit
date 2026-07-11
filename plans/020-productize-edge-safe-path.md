# Plan 020: Productize edge-safe consumer path in SPA + e2e docs

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- web/src/routes/api-clients.tsx e2e/ docs/`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: 018 (docs) recommended first
- **Category**: direction | dx
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`/v1/identify`, `resolve?tenantId=`, and ETag/304 are implemented and tested
in Go, but the admin SPA API Clients page only shows classic hostname resolve
curl snippets. Integrators copy the secret-heavy path for the edge.

## Current state

```tsx
// api-clients.tsx
const tenantResolveSnippet = `curl ... "/v1/resolve?hostname=<tenant-hostname>"`;
const resourceResolveSnippet = `curl ... "/v1/resolve/<tenant-hostname>/resources/<definition-key>"`;
```

E2E: `e2e/flows/admin-to-consumer-golden-path.md`,
`consumer-specific-resource-resolution.md`, persona `service-integrator.md`.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Typecheck | `cd web && bunx tsc --noEmit` | exit 0 |
| Unit | `cd web && bun run test` | exit 0 |

## Scope

**In**:
- `web/src/routes/api-clients.tsx` snippets (+ i18n if labels needed)
- `e2e/flows/*` and persona markdown updates
- Optional short `docs/developers/` integrator note if not covered by 018

**Out**: Changing HTTP handlers; implementing edge proxy code in this repo

## Git workflow

- Branch: `advisor/020-productize-edge-safe-path`
- Commit: `docs(web): edge identify + tenantId resolve snippets`
- No push/PR unless asked.

## Steps

### Step 1: Snippets on API Clients page

Add copyable examples:

1. `GET /v1/identify?hostname=...` (edge — no secrets)
2. `GET /v1/resolve?tenantId=<slug>` with `If-None-Match` comment
3. Keep hostname full resolve as “legacy / simple” secondary snippet

### Step 2: e2e + persona

Update golden path to include identify then resolve by tenantId.
Update service-integrator persona acceptance criteria.

**Verify**: typecheck; manually read markdown for consistency.

## Done criteria

- [x] SPA teaches identify + tenantId path
- [x] e2e/persona match code capabilities
- [x] README `DONE`

## STOP conditions

- Snippets would expose real tokens from env — use placeholders only.

## Maintenance notes

- Align with Cache-Control policy from plan 009 in prose.
