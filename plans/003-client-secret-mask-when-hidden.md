# Plan 003: Always mask secrets client-side when reveal mode is off

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the STOP conditions occurs, stop and report.
> When done, update `plans/README.md` for this plan unless the reviewer
> maintains the index.
>
> **Drift check (run first)**:
> `git diff --stat 21b541a..HEAD -- web/src/routes/tenant-detail.tsx web/src/components/ui/reveal-value.tsx web/src/components/ui/reveal-value.test.tsx web/src/lib/`
> On mismatch with "Current state", STOP.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (001 helps verification if already done)
- **Category**: security | bug
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Admin RN-06: secrets are masked unless explicitly revealed. The tenant detail
page loads cleartext when `?reveal=true`, then “hide secrets” sets
`secretsRevealed = false` **before** reloading masked data. While
`secretsRevealed` is false, the render path uses raw `<code>{f.value}</code>`,
so cleartext passwords remain (or flash) in the DOM — and stay if `load`
fails. Operators believe secrets are hidden when they are not.

## Current state

`web/src/routes/tenant-detail.tsx` — toggle:

```tsx
async function toggleSecrets() {
  const next = !secretsRevealed;
  setSecretsRevealed(next);
  toast.success(next ? t("tenantDetail.secretsEnabled") : t("tenantDetail.secretsDisabled"));
  await load(next);
}
```

Render branch (~336–340):

```tsx
{f.isSecret && secretsRevealed ? (
  <RevealValue hideLabel={t("common.hide")} showLabel={t("common.reveal")} value={f.value} />
) : (
  <code className="text-xs">{f.value || "—"}</code>
)}
```

When `secretsRevealed` is false but `f.value` is still the previous cleartext
from a reveal load, the else branch paints the password.

`web/src/components/ui/reveal-value.tsx` — local mask `••••••••••••` when
toggle off (per-field). Server mask constant is `••••••••` in
`server/internal/service/values.go` (`MaskedValue`).

`web/src/components/ui/reveal-value.test.tsx` — pattern for RTL + vitest tests.

**Conventions**: Prefer reusing `RevealValue` or a shared mask constant; use
existing i18n keys; keep toast UX.

## Commands you will need

| Purpose | Command | Expected |
|---------|---------|----------|
| Typecheck | `cd web && bunx tsc --noEmit` | exit 0 |
| Unit tests | `cd web && bun run test` | exit 0 |
| Gate (if 001 landed) | `make test-web` | exit 0 |

## Scope

**In scope**:
- `web/src/routes/tenant-detail.tsx`
- Optionally extract a tiny helper under `web/src/components/ui/` or
  `web/src/lib/` if it keeps the route cleaner
- New or extended tests under `web/src/**/*.test.tsx` (prefer testing a pure
  helper or small presentational component rather than the whole route)

**Out of scope**:
- Changing server mask string or `?reveal=true` API contract
- Full rewrite/split of `tenant-detail.tsx` (god route) beyond the secret display
- Other routes

## Git workflow

- Branch: `advisor/003-client-secret-mask-when-hidden`
- Commit: `fix(web): mask secrets when reveal mode is off`
- Do NOT push/PR unless asked.

## Steps

### Step 1: Fix render contract

**Rule**: if `f.isSecret && !secretsRevealed`, **never** render `f.value` —
always render a fixed mask (match `RevealValue` mask `••••••••••••` or export
a shared `SECRET_MASK` constant used by both).

Recommended render:

```tsx
{f.isSecret ? (
  secretsRevealed ? (
    <RevealValue
      hideLabel={t("common.hide")}
      showLabel={t("common.reveal")}
      value={f.value}
    />
  ) : (
    <code className="text-xs">{SECRET_MASK}</code>
  )
) : (
  <code className="text-xs">{f.value || "—"}</code>
)}
```

Optional hardening on hide (recommended):

In `toggleSecrets`, when turning reveal **off**:

1. Immediately clear secret field values in React state to the mask (or empty)
   **or** set a “pending hide” flag and keep showing mask until `load(false)`
   completes.
2. Then `await load(false)` to sync with server-masked values.

Simplest correct approach: **mask in render always when `!secretsRevealed`**,
so state content is irrelevant for display. Still call `load(false)` so state
does not retain cleartext longer than needed — optionally scrub secrets from
state when hiding:

```ts
setResources((prev) =>
  prev.map((r) => ({
    ...r,
    fields: r.fields.map((f) =>
      f.isSecret ? { ...f, value: SECRET_MASK } : f,
    ),
  })),
);
```

**Verify**: `cd web && bunx tsc --noEmit` → exit 0

### Step 2: Export shared mask if needed

If both `RevealValue` and tenant-detail hardcode different strings, unify:

- Export `SECRET_MASK = "••••••••••••"` from `reveal-value.tsx` (or `lib/secrets.ts`)
- Use it in `RevealValue` and tenant-detail

**Verify**: `rg -n "••••" web/src` → one shared definition preferred (or two
identical constants only if extraction is awkward — document why).

### Step 3: Tests

Add a focused unit test. Options (pick one):

**A (preferred):** pure function

```ts
// web/src/lib/secret-display.ts
export function displaySecretValue(opts: {
  isSecret: boolean;
  revealed: boolean;
  value: string;
  mask?: string;
}): string
```

Cases:

1. `isSecret=true, revealed=false, value="hunter2"` → mask, never `"hunter2"`
2. `isSecret=true, revealed=true, value="hunter2"` → `"hunter2"`
3. `isSecret=false, revealed=false, value="host"` → `"host"`

Use that function in the route render path.

**B:** extend `reveal-value.test.tsx` only if you keep logic inside components;
then add a tiny `SecretFieldValue` component with RTL tests asserting that with
`revealed={false}` the document does not contain the cleartext even when
`value` is cleartext.

**Verify**: `cd web && bun run test` → exit 0; new cases pass.

## Test plan

| Case | Expected |
|------|----------|
| Secret + reveal off + cleartext in props | DOM shows mask only |
| Secret + reveal on | cleartext available via RevealValue |
| Non-secret | always shows value |

Pattern: `web/src/components/ui/reveal-value.test.tsx`.

## Done criteria

- [x] With `secretsRevealed === false`, no secret cleartext is rendered for
      secret fields even if React state still holds cleartext
- [x] Unit test locks the regression
- [x] `bunx tsc --noEmit` and `bun run test` exit 0
- [x] No out-of-scope files
- [x] `plans/README.md` 003 → `DONE`

## STOP conditions

- Reveal UX is redesigned product-wide mid-change — STOP rather than inventing
  a new auth model.
- You need to change backend reveal semantics to make the UI safe — backend is
  already correct; fix is client-side.

## Maintenance notes

- Reviewer: check there is no path that renders `f.value` for secrets when
  `!secretsRevealed`.
- Future: prefer not keeping cleartext in list state when hidden (scrub on hide).
- Server remains source of truth for mask on fresh loads (`?reveal` omitted).
