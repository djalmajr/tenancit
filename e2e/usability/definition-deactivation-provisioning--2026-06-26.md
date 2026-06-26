# Usability — Desativação de definition afeta o provisionamento (definition-deactivation-provisioning)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — invariante cross-screen correto, **nenhum achado**
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1. **auth** — painel acessível (sessão já autenticada). ✅
2. **definitions-list** — criada definition `toggle-e2e` (Toggle E2E) ativa + campo obrigatório `host`. ✅
3. **tenant-detail (Acme Corp)** — "Adicionar recurso" → seletor "Tipo de recurso" lista **Dup Key E2E** e **Toggle E2E** (ambas ativas). ✅
4. **definition-detail** — `toggle-e2e` → **Desativar** → feedback "Definição desativada", botão vira **Ativar**. ✅
5. **tenant-detail** — "Adicionar recurso" novamente → seletor mostra **apenas Dup Key E2E**; `toggle-e2e` (inativa) **sumiu**. ✅
6. **definition-detail** — `toggle-e2e` → **Ativar** → feedback "Definição ativada", botão volta a **Desativar**. ✅
7. **tenant-detail** — "Adicionar recurso" → seletor volta a listar **Dup Key E2E** e **Toggle E2E**. ✅
8. **overview** — KPI **Definições ativas**: 2 (ambas ativas) → **1** (após desativar) → **2** (após reativar). ✅ (confirmado também via `GET /v1/admin/overview`).

## Findings (prioritized)
Nenhum. O invariante é consistente entre o detalhe da definition, o seletor de recurso do tenant e o KPI da visão geral — exatamente como o filtro `status === "active" && !activeKeys.has(d.key)` (`tenant-detail.tsx`) prevê.

## Observações
- A alternância usa `PUT /v1/admin/resource-definitions/{id}/status`; feedback textual ("Definição ativada/desativada") é claro.
- Complementa `resource-definition-management` (CRUD isolado) validando o efeito cross-screen.
