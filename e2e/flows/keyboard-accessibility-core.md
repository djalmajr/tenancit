---
id: keyboard-accessibility-core
name: Operar o caminho principal apenas por teclado e leitor de tela
reference: web/src/components/app-shell.tsx
persona: accessibility
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `konvario_admin_dev`
  - existe ao menos um tenant para abrir o detalhe
design_refs:
  shell: "planning/konvario/proto/components/app-shell.js"
  tenant-detail: "planning/konvario/proto/routes/tenant-detail.js"
  api-clients: "planning/konvario/proto/routes/api-clients.js"
---

## Objetivo do usuário

Completar o caminho operacional principal usando apenas teclado (e rótulos acessíveis), sem ficar preso em foco nem encontrar controles sem nome.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Na tela de **Acesso administrativo**, o foco inicia no campo **Token** (autoFocus); usar o botão de mostrar/ocultar token (com rótulo acessível) e enviar com **Enter** → o login ocorre por teclado sem precisar do mouse.
2. (`shell`) Navegar pela sidebar via **Tab** → cada item tem rótulo acessível e foco visível; ativar um item com **Enter** navega para a seção; os menus de idioma e tema são abertos e operados por teclado.
3. (`tenants-list`) Abrir o diálogo **Novo tenant** por teclado → o foco move-se para dentro do diálogo, **Tab** circula entre os campos/botões (focus trap), **Esc** fecha e devolve o foco ao gatilho.
4. (`tenant-detail`) Alcançar por teclado os botões de **revelar segredo** e os botões destrutivos (desativar/remover) → todos têm `title`/`aria-label` legíveis; o diálogo de confirmação de remoção é alcançável e **Esc**/Cancelar funcionam.
5. (`api-clients`) Gerar uma chave e alcançar a região do **token gerado** por teclado → o token é focalizável e copiável; o botão de copiar tem rótulo.

## Resultado esperado

Todo o caminho crítico é operável por teclado: o foco é gerenciado nos diálogos (trap + restauração), todos os controles interativos têm nome acessível, e nenhum elemento é uma armadilha de foco ou inalcançável.

## Estado atual × design

- O `AppShell` e os componentes de UI usam `aria-label`/`title` amplamente (itens de navegação, toggles, ações de ícone).
- Os diálogos (`Dialog`, `Sheet`, `Dropdown`) vêm da camada `web/src/components/ui/`; este fluxo valida o gerenciamento real de foco e a presença de rótulos.
- Achados típicos: controle só com ícone sem rótulo, foco que escapa do diálogo, ou ordem de tabulação confusa.
