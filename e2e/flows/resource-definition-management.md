---
id: resource-definition-management
name: Criar e manter resource definition
reference: docs/produto/03-jornadas-operacionais.adoc#criar-um-novo-tipo-de-recurso
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - usar uma key única por rodada, por exemplo `postgres-e2e`
design_refs:
  definitions-list: "planning/tenancit/proto/routes/definitions-list.js"
  definition-detail: "planning/tenancit/proto/routes/definition-detail.js"
---

## Objetivo do usuário

Criar um tipo de recurso dinâmico, definir campos obrigatórios/secretos e controlar seu status.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`definitions-list`) Autenticar-se se necessário e clicar em **Recursos** na sidebar → a grade de definitions aparece.
2. (`definitions-list`) Clicar em **Nova definição** → o diálogo abre com campos Key, Nome e Descrição.
3. (`definitions-list`) Preencher Key, Nome e Descrição e clicar em **Criar definição** → a aplicação navega para o detalhe da nova definition.
4. (`definition-detail`) Clicar em **Novo campo** → o diálogo **"Novo campo"** abre.
5. (`definition-detail`) Preencher Chave `host`, Label `Host`, manter tipo `string`, marcar **Obrigatório** e clicar em **Adicionar campo** → o campo aparece na tabela e a tela mostra feedback de campo adicionado.
6. (`definition-detail`) Clicar em **Novo campo** novamente, preencher Chave `password`, Label `Password`, marcar **Obrigatório** e **Segredo** e clicar em **Adicionar campo** → o campo aparece com checks em Obrigatório e Segredo.
7. (`definition-detail`) Clicar em **Desativar** → o status da definition muda para inactive, a ação muda para Ativar e a tela mostra feedback de desativação.
8. (`definition-detail`) Clicar em **Ativar** → o status volta para active e a tela mostra feedback de ativação.
9. (`definition-detail`) Clicar na lixeira de um campo → o diálogo de confirmação informa o campo afetado; confirmar **Remover** remove o campo e exibe feedback.
10. (`definitions-list`) Voltar para **Recursos** pela breadcrumb → o card da definition mostra contagem atualizada de campos e segredos.

## Resultado esperado

O operador consegue criar uma definition, adicionar campos dinâmicos, identificar campos obrigatórios/secretos, alternar status e remover campos pela UI.

## Estado atual × design

- A lista e o detalhe existem em `web/src/routes/definitions.tsx` e `web/src/routes/definition-detail.tsx`.
- A implementação atual usa ícones inferidos pela key; o IconPicker do protótipo não está presente na SPA real.
- Remoção de campo usa confirmação dedicada antes da ação destrutiva.
