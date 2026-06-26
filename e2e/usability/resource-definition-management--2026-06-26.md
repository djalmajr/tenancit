# Usability — Criar e manter resource definition (resource-definition-management)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (re-run contra o KonvarIO atual)
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433). Key de teste: `postgres-e2e`.

## Walkthrough
1–3. Recursos → **Nova definição** (Key/Nome/Descrição) → **Criar definição** → navega para o detalhe (Postgres E2E). ✅
4–5. **Novo campo** `host` (Label Host, tipo string, **Obrigatório**) → aparece na tabela, **toast "Campo adicionado."**. ✅
6. **Novo campo** `password` (**Obrigatório** + **Segredo**) → aparece com indicação de Segredo, **toast "Campo adicionado."**. ✅
7. **Desativar** → status inactive, ação vira **Ativar**, **toast "Definição desativada."**. ✅
8. **Ativar** → status active, **toast "Definição ativada."**. ✅
9. **Lixeira** de um campo → **confirm dialog nomeando o campo** → **Remover** → **toast "Campo removido."**. ✅
10. Breadcrumb → **Recursos** → card "Postgres E2E" mostra **"Campos: 1 · Segredos: 1"** (contagem atualizada). ✅

## Findings
Nenhum. **Regressões verificadas OK:** diálogos (criar definição / novo campo) com foco, todos os feedbacks de sucesso agora **toasts sonner**, e a confirmação dedicada para remoção de campo intacta.
