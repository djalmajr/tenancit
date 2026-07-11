# Plan 019: Wire hard-delete tenant in the admin SPA

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- web/src/lib/api.ts web/src/routes/tenant-detail.tsx web/src/lib/i18n.tsx server/internal/httpapi/admin_actions.go`

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (irreversible cascade)
- **Depends on**: none (API already exists)
- **Category**: direction | feature
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

`DELETE /v1/admin/tenants/{id}` cascades domains/resources/values (API + SQL
FK). SPA only soft-deactivates via status — no `deleteTenant` in `api.ts`.
Operators cannot retire tenants from the panel; dead rows accumulate or people
use raw SQL.

## Current state

- API: `admin_actions.go` `deleteTenant` → 204 / 404
- SPA: `api.ts` has delete domain/resource/field, not tenant
- `tenant-detail.tsx` has status edit + confirm dialogs for other deletes
- Soft vs hard should be explicit in UI copy

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Typecheck | `cd web && bunx tsc --noEmit` | exit 0 |
| Unit | `cd web && bun run test` | exit 0 |
| API test already exists or add Go test | `cd server && go test ./internal/httpapi/ -count=1` | exit 0 |

## Scope

**In**:
- `web/src/lib/api.ts` `deleteTenant`
- `web/src/routes/tenant-detail.tsx` danger zone + confirm (type slug to
  confirm recommended)
- i18n strings pt/en/es in `i18n.tsx`
- Optional e2e flow note under `e2e/flows/tenant-management.md`

**Out**: Soft-delete-only product change; recycle bin; undo

## Git workflow

- Branch: `advisor/019-spa-tenant-hard-delete`
- Commit: `feat(web): hard-delete tenant with cascade confirm`
- No push/PR unless asked.

## Steps

### Step 1: API client

```ts
deleteTenant: (id: string) =>
  req(`/tenants/${id}`, { method: "DELETE" }),
```

### Step 2: UI

- Danger zone on tenant detail: “Delete tenant permanently”
- Confirm dialog listing cascade (domains, resources, secrets)
- Prefer requiring user to type tenant slug before enabling confirm
- On success: toast + navigate to `/tenants`
- Use existing `ConfirmDialog` patterns on the page

### Step 3: i18n

Add keys for all three locales.

### Step 4: Manual / test

If easy, unit-test api method path; otherwise typecheck only.

**Verify**: `bunx tsc --noEmit` && `bun run test`

## Done criteria

- [x] Operator can hard-delete from SPA with strong confirm
- [x] Soft inactive remains available and distinct in copy
- [x] README `DONE`

## STOP conditions

- Product decides hard-delete must be disabled in prod — STOP.
- API returns non-204 unexpectedly — fix against server contract first.

## Maintenance notes

- Pair with plan 002 so inactive stops resolve; delete purges data.
- Reviewer: no accidental click path; cascade copy accurate.
