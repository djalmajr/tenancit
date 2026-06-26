# Usability — Cadastrar e manter tenant com domínio (tenant-management)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (re-run contra o KonvarIO atual)
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1–2. Tenants → **Novo tenant**: diálogo abre e **recebe o foco** (fix desta sessão). ✅
3. Nome/Slug (`acme-e2e`) + **Criar tenant** → diálogo fecha e navega para o detalhe. ✅
4. **Editar** nome → header reflete "Acme E2E Editado" e feedback agora é **toast "Tenant atualizado." (success)**. ✅
5–6. Aba **Domínios** (vazia) → **Adicionar** `app.acme-e2e.local` → aparece na tabela, prontidão reflete, **toast "Domínio adicionado."**. ✅
7. **Tenants** + busca `acme-e2e` → filtra para exatamente 1 (Acme E2E Editado). ✅
8. Detalhe → Domínios → **lixeira** → **confirm dialog com o hostname** → **Remover** → domínio removido, **toast "Domínio removido."**, empty state "Nenhum domínio…" de volta. ✅

## Findings
Nenhum. **Regressões verificadas OK:** foco nos diálogos (criar/editar/adicionar), os feedbacks de sucesso agora são **toasts sonner** (não mais banners inline), e o fluxo destrutivo mantém a confirmação dedicada.
