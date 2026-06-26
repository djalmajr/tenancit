---
id: admin-invalid-token-recovery
name: Recuperar acesso após token administrativo inválido
reference: web/src/components/app-shell.tsx#admin-auth-required
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo de desenvolvimento conhecido: `tenancit_admin_dev`
design_refs:
  auth: "planning/tenancit/proto/components/app-shell.js"
  overview: "planning/tenancit/proto/routes/overview.js"
---

## Objetivo do usuário

Entender que o token informado é inválido, corrigir o valor e recuperar o acesso sem precisar manipular storage, recarregar manualmente ou adivinhar caminhos.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Acessar o painel sem token administrativo salvo → a tela dedicada **"Acesso administrativo"** aparece com campo **Token** e botão **Entrar**.
2. (`auth`) Preencher **Token** com `rt_admin_invalido` e clicar em **Entrar** → a aplicação tenta autenticar, rejeita o token e retorna para a tela de acesso sem renderizar o painel.
3. (`auth`) Observar a mensagem abaixo do título → a tela informa que é necessário token administrativo para acessar o painel.
4. (`auth`) Preencher **Token** com `tenancit_admin_dev` e clicar em **Entrar** → a visão geral carrega com KPIs reais e sidebar visível.
5. (`overview`) Clicar em **Sair** → o acesso é removido e a tela de **Acesso administrativo** volta a bloquear o painel.

## Resultado esperado

O operador recebe feedback claro para token inválido, consegue corrigir a credencial pela própria UI e o painel não fica preso em estado parcialmente autenticado.

## Estado atual × design

- O fluxo valida o evento global `admin-auth-required`, disparado quando a Admin API retorna `401`.
- A recuperação deve acontecer pela tela de login; manipular `localStorage` diretamente seria um desvio do fluxo real do usuário.
