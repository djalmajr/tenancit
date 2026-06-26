---
id: i18n-and-preferences-persistence
name: Tradução completa das telas e persistência de idioma e tema
reference: web/src/lib/i18n.tsx
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - existe ao menos um tenant e uma definition para inspecionar telas de detalhe (ou criar durante o fluxo)
design_refs:
  shell: "planning/tenancit/proto/components/app-shell.js"
  overview: "planning/tenancit/proto/routes/overview.js"
---

## Objetivo do usuário

Operar o painel no idioma preferido (Português, English ou Español) com todas as telas traduzidas, e ter idioma e tema preservados entre navegação, recarga e logout.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Autenticar-se com `tenancit_admin_dev` → visão geral carrega.
2. (`shell`) Abrir o dropdown de idioma no header e escolher **English** → navegação, header e visão geral mudam para inglês.
3. (`shell`) Percorrer **Tenants**, abrir um tenant (**tenant-detail**), **Recursos** (**definitions-list**), abrir uma definition (**definition-detail**) e **Chaves de API** (**api-clients**) → revisar cada tela e registrar qualquer texto que permaneça em português (rótulos, títulos de diálogo, mensagens, botões). Tokens de exemplo, snippets `curl` e os códigos de tipo (`string`/`int`/`bool`) são literais por design e não contam como falha.
4. (`shell`) Trocar o idioma para **Español** → revisar visão geral, tenants e chaves de API e confirmar a tradução equivalente.
5. (`shell`) Abrir o dropdown de tema e escolher **Escuro** → o shell aplica o tema sem perder contraste/legibilidade; testar também **Sistema**.
6. (`overview`) Recarregar a página no navegador (refresh) → o idioma e o tema escolhidos permanecem (persistidos em `tenancitLocale` e `tenancitTheme`).
7. (`shell`) Clicar em **Sair** → a tela de **Acesso administrativo** volta, e os controles de idioma/tema do login refletem a preferência persistida.

## Resultado esperado

Toda a interface segue o idioma selecionado em todas as telas, sem strings hardcoded remanescentes; o tema é aplicado de forma consistente (incluindo escuro); e idioma + tema sobrevivem à recarga e reaparecem na tela de login. Qualquer texto não traduzido é um achado de i18n.

## Estado atual × design

- O dicionário de tradução vive em `web/src/lib/i18n.tsx` com `pt`, `en` e `es`; o tema em `web/src/lib/theme.tsx`.
- As preferências são guardadas em `localStorage` sob `tenancitLocale` e `tenancitTheme`.
- Os controles de idioma e tema aparecem tanto na tela de login quanto no header autenticado.
