# Usability — Admin até Consumer API por hostname (admin-to-consumer-golden-path)
- **Persona:** service-integrator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (fluxo híbrido; validação assistida por terminal)
- **Ambiente:** stack Tenancit atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
Os blocos de configuração no painel (criar definition + campos, tenant, domínio, recurso, API key) foram validados em detalhe nos flows irmãos desta mesma rodada — `resource-definition-management`, `tenant-management`, `tenant-resource-lifecycle`, `api-client-token-lifecycle` (todos ✅). Aqui o foco é o **contrato da Consumer API** ponta a ponta.

Setup: tenant `gp-tenant-e2e`, domínio `gp.e2e.local`, recurso `dbconn-e2e` (host + password segredo), API key `gp-consumer-key`.

- **Step 7** — `GET /v1/resolve?hostname=gp.e2e.local` com `Authorization: Bearer <token>` → **200**, retorna `{"tenantSlug":"gp-tenant-e2e","resources":[{"definitionKey":"dbconn-e2e","values":{"host":"db.gp.local","password":"gp-secret-123"}}]}` — slug do tenant + recurso ativo com **segredo descriptografado** sobre o contrato HTTP. ✅
- **Step 8** — sem token → **401**. ✅
- **Step 9** — hostname desconhecido + token válido → **404**. ✅

## Findings
Nenhum. O caminho admin→consumer funciona de ponta a ponta; segredo entregue descriptografado apenas para chamada autenticada; erros 401/404 corretos.
