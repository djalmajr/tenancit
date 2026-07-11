# Plan 021: Spike design — admin action audit log (roadmap P1)

> **Executor instructions**: This is a **design/spike plan**, not a full
> implementation. Produce a short design doc + optional spike PR notes.
> Do not build the entire audit product unless the design is approved mid-plan.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- docs/business/04-escopo-e-roadmap.adoc server/migrations/ docs/adr/`

## Status

- **Execution status**: **DONE (design spike)** — output:
  `docs/developers/design/admin-audit-log.md`
- **Execution priority**: P3 spike (product initiative remains roadmap P1)
- **Effort**: L for full build; **this plan = S–M spike only**
- **Risk**: MED if implemented naively
- **Depends on**: none; before human IdP login ideally
- **Category**: direction
- **Planned at**: commit `21b541a`, 2026-07-08

## Why this matters

Roadmap P1 (`docs/business/04-escopo-e-roadmap.adoc`): audit who changed what
and when. Shared `TENANCIT_ADMIN_TOKEN` (ADR 0004) means no human actor today.
Without audit, secret reveals and deletes are unattributable.

## Current state

- No audit table in `00001_init.sql`
- Admin mutations unlogged beyond process stdout
- `?reveal=true` is sensitive (RN-06)

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Spike output | design markdown exists | under `docs/` or `plans/021-notes.md` |

## Scope

**In**:
- Design document: event schema, actor model under shared token, which
  endpoints to instrument first (reveal, delete tenant, create API client,
  provision resource), retention, query API sketch
- Optional: minimal proof (one table + log one endpoint) **only if** design
  fits in same PR and is clearly labeled experimental

**Out**: Full multi-operator IdP; SIEM export; immutable legal hold

## Git workflow

- Branch: `advisor/021-admin-audit-log-spike`
- Commit: `docs: admin audit log design spike`
- No push/PR unless asked.

## Steps

### Step 1: Read constraints

- ADR 0004 admin token
- Roadmap P1 audit + login human
- List admin routes in `server.go`

### Step 2: Write design (`docs/developers/08-audit-log-design.adoc` or
`docs/adr/0005-admin-audit-log.md` draft status Proposed)

Must answer:

1. Actor identity until IdP: constant `admin-token` vs request header
   `X-Actor-Email` (untrusted) vs future OIDC `sub`
2. Event row: `id, at, actor, action, target_type, target_id, meta jsonb, request_id`
3. Which actions are in v1
4. Whether reveal logs **that** reveal happened, never the secret value
5. Read API: admin-only list with filters
6. Migration strategy and volume estimate

### Step 3: Explicit non-goals and open questions

**Verify**: design file merged; no production secrets in examples.

## Done criteria

- [x] Design doc created and reviewable
- [x] Open questions listed
- [x] README status `DONE` (spike) — full implementation is a follow-up plan

The index update is intentionally left to the orchestration step that reconciles
all plan statuses. No production implementation was added by this spike.

## STOP conditions

- Implementing full audit UI without design buy-in — stop after design unless
  operator explicitly expands scope.

## Maintenance notes

- After IdP (roadmap), map actor to real users; keep token as break-glass actor.
