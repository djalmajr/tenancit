---
id: responsive-mobile-navigation
name: Navegação e operação do painel em viewport mobile
reference: web/src/components/ui/sidebar.tsx
persona: mobile
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `konvario_admin_dev`
  - navegador em largura mobile (ex.: 390px) durante todo o fluxo
  - existe ao menos um tenant e uma definition para abrir telas de detalhe
design_refs:
  shell: "planning/konvario/proto/components/app-shell.js"
  tenant-detail: "planning/konvario/proto/routes/tenant-detail.js"
  definitions-list: "planning/konvario/proto/routes/definitions-list.js"
---

## Objetivo do usuário

Usar o painel administrativo num celular e conseguir navegar, ler e operar todas as seções principais sem elementos cortados, inacessíveis ou sobrepostos.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Em largura mobile, acessar o painel e autenticar com `konvario_admin_dev` → o card de **Acesso administrativo** cabe na tela e os controles de idioma/tema permanecem acionáveis.
2. (`shell`) Observar a sidebar → em mobile ela fica recolhida/offcanvas; tocar no **gatilho de menu** (SidebarTrigger) → a navegação abre como painel/sheet sobre o conteúdo.
3. (`shell`) Tocar em **Tenants**, **Recursos** e **Chaves de API** pelo menu mobile → cada seleção navega e o menu se fecha, deixando o conteúdo visível.
4. (`tenants-list`) Conferir a tabela de tenants e a busca em mobile → o conteúdo reflui ou rola horizontalmente sem quebrar; **Novo tenant** continua acionável.
5. (`tenant-detail`) Abrir um tenant → cabeçalho, card de **Prontidão para consumo** (grid de 4 colunas) e as abas reflui para empilhar; abrir o diálogo **Adicionar recurso** → o diálogo cabe na viewport e é utilizável.
6. (`definitions-list`) Abrir **Recursos** → a grade de cards reflui para uma coluna.
7. (`api-clients`) Abrir **Chaves de API**, criar uma chave e revelar o token → o diálogo e o botão de copiar funcionam em mobile sem cortar o token.

## Resultado esperado

Nenhum elemento essencial fica inacessível ou cortado em mobile: a navegação por sheet funciona, tabelas e grids reflui/rolam, e os diálogos (incluindo o de token) são totalmente operáveis em tela estreita.

## Estado atual × design

- A sidebar usa `web/src/components/ui/sidebar.tsx` com `web/src/hooks/use-mobile.ts` para alternar para o modo Sheet em telas pequenas.
- Os layouts usam grids responsivos (`sm:grid-cols-2`, `lg:grid-cols-3/4`, `md:grid-cols-4`); este fluxo valida o comportamento real em viewport mobile.
