# Usability — Resolver recurso específico por hostname e key (consumer-specific-resource-resolution)
- **Persona:** service-integrator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (fluxo híbrido; validação assistida por terminal)
- **Ambiente:** stack TenancitIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
Blocos de configuração no painel validados nos flows irmãos desta rodada. Foco: o endpoint `/v1/resolve/{hostname}/resources/{definitionKey}`.

Setup: tenant `gp-tenant-e2e` (domínio `gp.e2e.local`) com **dois** recursos ativos — `dbconn-e2e` e `postgres-e2e`; API key ativa.

- **Step 7** — `GET /v1/resolve/gp.e2e.local/resources/dbconn-e2e` com token → **200**, retorna **somente** o recurso `dbconn-e2e` (controle: o resolve geral lista os dois — `['dbconn-e2e','postgres-e2e']`). ✅
- **Step 8** — `definitionKey` inexistente (`naoexiste-key`) + token válido → **404**. ✅
- **Step 9** — revogar a chave e repetir a chamada válida → **401**. ✅

## Findings
Nenhum. O endpoint distingue recurso específico (200 com 1 recurso), recurso inexistente (404) e autenticação inválida/revogada (401). Observação (do flow `api-client-token-lifecycle`): o endpoint específico não é citado na UI — possível orientação para integradores; o snippet visível em "Ajuda" cobre apenas `/v1/resolve?hostname=`.
