# Usability — Operar o caminho principal por teclado (keyboard-accessibility-core)
- **Persona:** accessibility · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ⚠️→✅ — 1 achado alto (focus-management dos diálogos), **corrigido e verificado**; 1 nota baixa
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1. **auth** — `document.activeElement` inicia no campo **Token** (autoFocus); botão com `aria-label="Mostrar token"`; login por **Enter** (form.requestSubmit) funciona sem mouse. ✅
2. **shell** — itens de navegação todos com nome acessível (`aria-label`); menus de idioma/tema abrem e operam por teclado. ✅
3. **diálogos** — **ACHADO**: o `Dialog` (implementação própria) não movia o foco para dentro ao abrir, **não prendia o Tab** (escapava para o fundo) e não restaurava o foco. ❌→✅ **corrigido**.
4. **tenant-detail / definition-detail** — varredura de TODAS as telas (overview, tenants, tenant-detail abas Recursos+Domínios, definitions, definition-detail, api-clients): **0 controles interativos sem nome acessível**. Botões de ícone/ação rotulados: "Habilitar revelação de segredos", "Remover" (domínio/campo), "Desativar"/"Ativar", "Editar", paginação ("Primeira página", "Nome: Ordenar decrescente", etc.). ✅
5. **api-clients** — diálogo de token gerado: token alcançável, botão **"Copiar"** rotulado, sem botões sem rótulo; o diálogo agora recebe foco. ✅

## Findings (prioritized)
| # | Severity | Step | What happened | Fix |
|---|---|---|---|---|
| 1 | high | 3 | O `Dialog` (`ui/dialog.tsx`, implementação própria com portal) tinha `role="dialog"`/`aria-modal`/Esc, mas **nenhum gerenciamento de foco**: ao abrir, o foco ficava no gatilho (fora do modal); o **Tab escapava** para o conteúdo atrás; e o foco não era restaurado ao fechar. Afetava todos os diálogos (criar tenant/definition, adicionar campo/domínio/recurso, token). | **CORRIGIDO** — `DialogContent` agora, ao abrir, move o foco para o 1º focável (ou o painel), **prende o Tab/Shift+Tab** dentro do diálogo, e **restaura o foco** ao elemento anterior no fechamento. Painel ganhou `tabIndex={-1}`. Re-teste: abrir "Novo tenant" → foco no campo Nome; Tab cicla preso; Esc fecha e devolve o foco ao botão "Novo tenant". Verificado também no diálogo de token. |
| 2 | low | 3 | O painel do diálogo tem `role="dialog"` + `aria-modal` mas **não** referencia o `DialogTitle` via `aria-labelledby`, então o leitor de tela pode não anunciar o título como nome acessível do diálogo. | Recomendado (não aplicado): ligar `aria-labelledby` ao id do `DialogTitle` (via `useId` no contexto do Dialog). |

## Observações
- Cobertura de `aria-label`/`title` é exemplar em ícones e ações; foco visível presente.
- O token é exibido em elemento `code` (não input focável) — o caminho acessível de cópia é o botão "Copiar" rotulado, que funciona por teclado.
