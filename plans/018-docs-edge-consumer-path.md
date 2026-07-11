# Plan 018: Document dual consumer path (identify + resolve by tenantId + ETag)

> **Executor instructions**: Follow step by step; verify; STOP on conditions.
> Update `plans/README.md` when done.
>
> **Drift check**: `git diff --stat 21b541a..HEAD -- docs/`

## Status

- **Status**: DONE
- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (pairs with 020 for SPA snippets)
- **Category**: docs
- **Planned at**: commit `21b541a`, 2026-07-08
- **Completed at**: 2026-07-09

## Why this matters

Code and `docs/developers/03-contratos-http.adoc` already describe
`/v1/identify`, `resolve?tenantId=`, and ETag/304. Business/architecture/
security/journey docs still show only classic
`GET /v1/resolve?hostname=…` with full secrets. Integrators following the
business path wire the edge to full secret resolution and miss the safer
pattern.

## Current state

- Complete: `docs/developers/03-contratos-http.adoc` permanece a fonte
  normativa para identify, tenantId, ETag, cache e erros.
- Complete: visão geral, glossário, jornada de borda sem segredos, sequência de
  arquitetura e fronteiras de segurança refletem o caminho dual.

## Commands

| Purpose | Command | Expected |
|---------|---------|----------|
| Grep consistency | `rg -n 'identify|tenantId|ETag' docs/` | hits in business + eng |

## Scope

**In**: AsciiDoc under `docs/` listed above (and small cross-links)

**Out**: Code changes; SPA (plan 020); rewriting ADRs

## Git workflow

- Branch: `advisor/018-docs-edge-consumer-path`
- Commit: `docs: dual consumer path identify + resolve tenantId`
- No push/PR unless asked.

## Steps

### Step 1: Architecture sequence

Update mermaid/flow in `01-arquitetura.adoc` to show:

1. Edge → `GET /v1/identify?hostname=` → `tenantSlug` (no secrets)
2. App → `GET /v1/resolve?tenantId=` with API key + optional If-None-Match

Keep hostname resolve as supported alternate.

### Step 2: Business journey

Add journey “borda sem segredos” in `03-jornadas-operacionais.adoc`.

### Step 3: Security table

List `/v1/identify` and `/v1/resolve*` under consumer API key auth in
`04-seguranca-e-criptografia.adoc`.

### Step 4: Glossary

Define identify vs resolve responsibilities in `02-glossario.adoc`.

**Verify**: no contradictory “only resolve by hostname” claims remain in those
files (`rg`).

## Done criteria

- [x] Business + architecture + security mention identify and tenantId resolve
- [x] contratos-http remains source of HTTP detail
- [x] README `DONE`

## STOP conditions

- Product deprecates identify — not the case at plan time.

## Maintenance notes

- Link e2e flows when plan 020 updates them.
