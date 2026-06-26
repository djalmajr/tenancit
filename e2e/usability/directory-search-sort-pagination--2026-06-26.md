# Usability — Busca, ordenação e paginação das tabelas (directory-search-sort-pagination)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — DataTable robusta, **nenhum achado**
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433); seed de 15 tenants e 14 API clients ativos.

## Walkthrough
2. **tenants-list** — 15 itens, "Página 1 de 3", 5 linhas/página, headers Nome/Slug/Status, controles de paginação visíveis. ✅
3. **paginação** — Próxima→"2 de 3", Última→"3 de 3" (Próxima/Última **disabled** no fim), Primeira→"1 de 3" (Primeira/Anterior **disabled** no início). Limites corretos, indicador atualiza. ✅
4. **linhas por página** — opções [5,10,25,50]; "10" → 10 linhas, "Página 1 de 2". ✅
5. **ordenação tri-state** — Nome: asc(default,"Acme Corp") → desc("Lima Corp") → reset; `aria-label` descreve a próxima ação ("Ordenar decrescente"/"Limpar ordenação"/"Ordenar crescente"). Slug e Status idem. ✅
6. **busca** — "charlie" (nome) e "delta" (slug) filtram corretamente; sem match → "Nenhum tenant encontrado"; limpar → conjunto completo. Placeholder "Buscar por nome ou slug...". ✅
7. **api-clients** — headers Nome/Token/Criado em/Status/Ações; "14 itens, Página 1 de 3"; default **Criado em desc** (svc-lima primeiro); **token mascarado** (`rt_live_••••`); busca "nome/token/status" e ordenação por coluna funcionam igual. ✅

## Findings (prioritized)
Nenhum. Paginação com limites e disabled-states corretos, ordenação tri-state por coluna, busca multi-campo com empty state, seleção de linhas-por-página, e mascaramento de token na listagem.

## Observações (polish, não-bloqueante)
- Placeholder da busca de tenants diz "nome ou slug"; o `globalFilterFn` (design) também considera status — a dica poderia mencionar status (consistência menor). Não afeta o uso.
- `aria-label`s das ações de paginação/ordenação são exemplares para acessibilidade (reaproveitados no flow keyboard-accessibility-core).
