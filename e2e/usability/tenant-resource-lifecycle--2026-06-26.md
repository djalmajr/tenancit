# Usability — Provisionar recurso e validar segredos (tenant-resource-lifecycle)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (re-run contra o KonvarIO atual)
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433). Seed: definition `dbconn-e2e` (DB Conn E2E) com campos `host` (obrigatório) e `password` (obrigatório, segredo); tenant `acme-e2e`.

## Walkthrough
1–2. Tenants → detalhe do tenant (aba Recursos). ✅
3. **Adicionar recurso** → seletor lista os tipos ativos disponíveis (DB Conn E2E, Dup Key E2E, Toggle E2E). ✅
4. Selecionar **DB Conn E2E** → campos dinâmicos: **Host *** (texto) e **Senha *** (`type=password`, segredo mascarado no form). ✅
5. **Salvar recurso** desabilitado até preencher os obrigatórios; preenchido → recurso na lista, prontidão "Recursos ativos", **toast "Recurso adicionado."**. ✅
6. Segredo **mascarado por padrão** (valor claro não aparece). ✅
7. **Habilitar revelação de segredos** → **toast "Revelação por campo habilitada."** + controle de revelar por campo; ainda mascarado até ação. ✅
8. Clicar no **revelar** do campo segredo → valor claro visível **só após ação explícita**. ✅
9–10. **Desativar** → **toast "Recurso desativado."**; **Reativar** → **toast "Recurso reativado."** (status volta a "ativo"). ✅
11. **Remover** → confirm dialog nomeando o recurso ("O recurso DB Conn E2E será removido…") → confirmar → **toast "Recurso removido."** + empty state "Nenhum recurso". ✅

## Findings
Nenhum. Cobre também o carry-over do flow `admin-form-validation-and-errors` (validação do form de recurso: obrigatório desabilita o save, campo `int`/secreto renderizam corretamente). **Regressões verificadas OK:** todos os feedbacks de sucesso agora são **toasts sonner**; máscara/revelação de segredo e confirmação destrutiva intactas.
