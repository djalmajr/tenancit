# Usability — Validação de formulários e erros de unicidade (admin-form-validation-and-errors)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ⚠️ completável com fricção → **achados corrigidos nesta sessão** (re-verificado ✅)
- **Ambiente:** stack Tenancit atual (Vite :5180 + API :8087 + Postgres vazio :5433)

## Walkthrough
- **Passo 2 (disabled):** "Novo tenant" com Nome/Slug vazios → botão **Criar tenant** desabilitado; ao preencher ambos, habilita. ✅
- **Passo 3 (slug duplicado):** criar `dup-e2e`, depois tentar criar de novo com o mesmo slug → falha 409, diálogo **permanece aberto** e **não navega**. ✅ estrutura — ❌ mensagem (texto cru).
- **Passo 4 (cancelar):** **Cancelar** fecha o diálogo sem criar nada. ✅
- **Passo 5 (key duplicada):** criar definition `dupkey-e2e`, tentar duplicar → não navega, mas o **erro aparecia FORA do diálogo** (banner de página atrás do modal) → sem feedback em contexto. ❌
- **Passo 6 (hostname duplicado):** aba Domínios, adicionar `dup.e2e.local`, tentar de novo → 409 **no diálogo**, sem navegação. ✅ estrutura — ❌ mensagem (texto cru).
- **Passos 7–8 (form de recurso):** não totalmente exercidos nesta rodada — o form de recurso exige uma definition com campos obrigatório/int/secreto; rótulo "Recursos" duplicado (sidebar vs aba do tenant) confundiu a navegação. Carry-over.

## Findings (prioritized)
| # | Severity | Step | What happened | Fix |
|---|---|---|---|---|
| 1 | high | 3,5,6 | Todo conflito de unicidade exibia texto cru ao operador: `Error: 409: {"error":"ERROR: duplicate key value violates unique constraint \"...\" (SQLSTATE 23505)"}` — incompreensível e vazando internals do Postgres. | **CORRIGIDO** — backend (`admin.go`) passou a devolver mensagens limpas (`a tenant with this slug already exists`, etc.); frontend ganhou `ApiError` tipado (`api.ts`) + `apiErrorMessage(e,t)` (`i18n.tsx`) que mapeia status→mensagem **localizada** (pt/en/es), nunca texto cru. Aplicado a todos os `setError`/`setPageError`/`setFieldError`. |
| 2 | high | 5 | O diálogo "Nova definição" renderizava o erro de criação **fora do diálogo** (nível de página, atrás do overlay) → operador sem feedback em contexto. | **CORRIGIDO** — erro renderizado **dentro** do `DialogContent` (`definitions.tsx`); limpo ao abrir/fechar o diálogo. |
| 3 | low | nav | Sidebar "Recursos" (→ `/resource-definitions`) usa o mesmo rótulo da aba "Recursos" do tenant-detail, e o título da página é "Definições de recurso" — rótulo de nav inconsistente com o título. | Aberto (não corrigido) — considerar renomear a nav para "Definições". |
| 4 | info | 7–8 | Validação do form de recurso (mensagem "nenhum tipo disponível", desabilitar com obrigatório vazio, mascarar secreto) não totalmente exercitada — requer definition com campos. | Carry-over para `tenant-resource-lifecycle` / próxima rodada. |

## Verificação dos fixes (re-run)
- Tenant slug duplicado → diálogo mostra **"Já existe um registro com esse valor. Verifique se não está duplicando."** (localizado, sem SQL). ✅
- Definition key duplicada → mesma mensagem **dentro do diálogo**; banner de página limpo ao cancelar. ✅
- Backend `POST /v1/admin/tenants` (slug duplicado) → `{"error":"a tenant with this slug already exists"}` (sem SQL). ✅
- `tsc --noEmit` e `go build ./...` verdes. ✅

## Arquivos tocados pelo fix
- `server/internal/httpapi/admin.go` (3 mensagens de conflito limpas)
- `web/src/lib/api.ts` (`ApiError`), `web/src/lib/i18n.tsx` (`apiErrorMessage` + chaves `errors.*`)
- `web/src/routes/{tenants,definitions,definition-detail,api-clients,index}.tsx` (uso de `apiErrorMessage`; erro no diálogo de definitions)
