---
id: admin-auth-overview
name: Acessar painel administrativo e revisar visão geral
reference: docs/business/03-jornadas-operacionais.adoc#jornadas-operacionais
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento (`make dev-compose` ou equivalente)
  - API admin acessível pelo proxy do Vite em `/v1/admin`
  - token administrativo de desenvolvimento conhecido: `tenancit_admin_dev`
design_refs:
  overview: "planning/tenancit/proto/README.md#telas-rotas"
  shell: "planning/tenancit/proto/components/app-shell.js"
---

## Objetivo do usuário

Acessar o painel, autenticar-se quando necessário e entender rapidamente o estado operacional do serviço.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`overview`) Acessar o painel sem token administrativo salvo → a tela dedicada **"Acesso administrativo"** aparece em card central, sem exibir sidebar ou dashboard por trás.
2. (`overview`) Usar o dropdown de idioma no login alternando entre **Português**, **English** e **Español** → o trigger fechado exibe apenas a bandeira, o menu aberto exibe bandeira e nome nativo, e título, descrição, botão e rótulos mudam para o idioma selecionado.
3. (`overview`) Usar o dropdown de tema no login alternando entre **Claro**, **Escuro** e **Sistema** → a tela aplica a preferência sem deslocar o card.
4. (`overview`) Preencher o campo **Token** com `tenancit_admin_dev` e clicar em **Entrar** → a página recarrega com os dados da visão geral e mantém idioma/tema escolhidos.
5. (`overview`) Conferir a primeira linha de indicadores → a tela exibe **Tenants ativos**, **Domínios**, **Recursos provisionados** e **Definições ativas**.
6. (`overview`) Conferir a segunda linha de indicadores → a tela resume **Saúde operacional**, **Requisições no mês**, **Chaves expirando** e **Dead letters**, sem repetir um diretório de tenants.
7. (`shell`) Usar o dropdown de idioma no header alternando entre **Português**, **English** e **Español** → navegação, header e conteúdo da visão geral mudam para o idioma selecionado.
8. (`shell`) Usar o dropdown de tema no header alternando entre **Claro**, **Escuro** e **Sistema** → o shell aplica a preferência sem perder contraste ou legibilidade.
9. (`shell`) Clicar nos itens da sidebar **Visão geral**, **Tenants**, **Recursos** e **Chaves de API** → cada item leva à tela correspondente e o título do header acompanha a seção.
10. (`shell`) Usar o botão de colapsar/expandir sidebar → a navegação continua utilizável por ícones/tooltips e o conteúdo principal não perde acesso.
11. (`shell`) Clicar no botão **Sair** no rodapé da sidebar → o token administrativo é removido, a página recarrega e a tela dedicada **"Acesso administrativo"** volta a bloquear o painel.

## Resultado esperado

O operador consegue autenticar-se, escolher idioma, alternar tema, ver indicadores de inventário e pulso operacional, navegar pelas seções principais, encerrar a sessão administrativa e reconhecer o estado inicial sem acessar URLs manualmente.

## Estado atual × design

- A tela existe em `web/src/routes/index.tsx` e consome `GET /v1/admin/overview`.
- A tela de token é global no `AppShell` e substitui o shell quando não há token administrativo salvo.
- Os controles de idioma e tema aparecem no login e no header autenticado.
- O protótipo cobre visão geral e navegação lateral; este fluxo valida paridade funcional, não pixel-perfect.
