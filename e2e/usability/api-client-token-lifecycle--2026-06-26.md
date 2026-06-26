# Usability — Gerar, copiar, revogar e reativar chave de API (api-client-token-lifecycle)
- **Persona:** service-integrator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ⚠️→✅ — 1 achado (reativar inalcançável) **corrigido e verificado**; 1 observação (doc desatualizado)
- **Ambiente:** stack Tenancit atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1. Chaves de API → tabela de chaves (token mascarado). O **alerta de segredos em claro** + **snippet `/v1/resolve`** ficam atrás do botão **"Ajuda"** (popover), não inline. ✅ (com observação)
2. **Ajuda → Copiar** o snippet → **toast "Snippet copiado"**. ✅
3. **Nova chave** → diálogo abre e **recebe foco**. ✅
4. Nome + **Gerar token** → diálogo mostra o **token completo** (62 chars). ✅
5. **Copiar** → botão indica **"Copiado"**. ✅
6. **Concluir** → diálogo fecha, chave na tabela com **token mascarado**. ✅
7. **Revogar** → **toast "Chave revogada."**. ✅
8. **Reativar** → ❌→✅ **estava inalcançável**; **corrigido**: chave revogada agora visível com ação **Reativar** → **toast "Chave reativada."**, status volta a "ativo". ✅

## Findings (prioritized)
| # | Severity | Step | What happened | Fix |
|---|---|---|---|---|
| 1 | medium | 8 | A lista filtrava `clients.filter(status==="active")`, então uma chave **revogada sumia** da tabela. Não havia ação **Reativar** implementada (só `revoke`), apesar de existirem as strings i18n (`apiClients.reactivate`, `statusReactivated`), o badge `destructive` para status não-ativo, e suporte no backend. Resultado: capacidade de reativação **inalcançável pela UI** (passo 8 bloqueado). | **CORRIGIDO** (`api-clients.tsx`): a tabela passa a mostrar **todas** as chaves; a coluna Ações é condicional (Revogar para ativas, **Reativar** para revogadas) e foi adicionada a função `reactivate` (`setAPIClientStatus(id,"active")` + toast `statusReactivated`). |
| 2 | low | 1 | O flow doc descreve o alerta de segredos + snippet `/v1/resolve` como visíveis inline; na UI atual eles ficam atrás do botão **"Ajuda"** (popover). Info clara a 1 clique — não é bug, mas o doc está desatualizado. | Anotado (doc); considerar deixar o alerta de segurança mais proeminente. |

## Verificação do fix
`tsc` ✅ · web 17/17 ✅ · re-run: chave `svc-lifecycle-e2e` revogada → reaparece com badge "revogado" + ação Reativar → reativada com toast e status "ativo". ✅
