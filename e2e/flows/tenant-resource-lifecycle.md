---
id: tenant-resource-lifecycle
name: Provisionar recurso do tenant e validar segredos
reference: docs/produto/03-jornadas-operacionais.adoc#revelar-segredos-no-painel
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - existe um tenant ativo
  - existe uma resource definition ativa ainda não provisionada para esse tenant, com ao menos um campo obrigatório e um campo segredo
design_refs:
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
  definition-detail: "planning/tenancit/proto/routes/definition-detail.js"
---

## Objetivo do usuário

Provisionar um recurso para um tenant, confirmar mascaramento de segredo e controlar o ciclo de vida do recurso.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`tenant-detail`) Autenticar-se se necessário e clicar em **Tenants** na sidebar → a lista de tenants aparece.
2. (`tenant-detail`) Clicar no tenant de teste → o detalhe abre na aba Recursos.
3. (`tenant-detail`) Revisar **Prontidão para consumo** e clicar em **Adicionar recurso** → o diálogo mostra tipos ativos disponíveis ou informa que todos já estão provisionados.
4. (`tenant-detail`) Selecionar a resource definition disponível → os campos dinâmicos da definition aparecem no formulário.
5. (`tenant-detail`) Preencher todos os campos obrigatórios, incluindo o campo segredo, e clicar em **Salvar recurso** → o diálogo fecha, o recurso aparece na lista, a prontidão indica recurso ativo e a tela mostra feedback.
6. (`tenant-detail`) Verificar o valor do campo segredo sem revelar segredos → o valor permanece mascarado.
7. (`tenant-detail`) Clicar em **Habilitar revelação de segredos** → a lista recarrega com controles de revelação por campo e a tela informa que a revelação por campo foi habilitada.
8. (`tenant-detail`) Clicar no ícone de revelar do campo segredo → o valor claro fica visível apenas após a ação explícita.
9. (`tenant-detail`) Clicar em **Desativar** no recurso → o status do recurso muda para inactive, a ação passa a reativar e a tela mostra feedback.
10. (`tenant-detail`) Clicar em **Reativar** no recurso → o status volta para active e a tela mostra feedback.
11. (`tenant-detail`) Clicar em **Remover** no recurso → o diálogo de confirmação informa o recurso afetado; confirmar **Remover** remove o recurso, mostra feedback e, se não houver outros, a tela mostra o estado vazio.

## Resultado esperado

O recurso respeita campos dinâmicos da definition, segredos são mascarados por padrão, revelação exige ação explícita e o operador consegue desativar, reativar e remover o recurso.

## Estado atual × design

- O fluxo está implementado em `web/src/routes/tenant-detail.tsx`.
- A regra de 1 recurso ativo por definition é reforçada pelo diálogo: definitions já ativas no tenant deixam de aparecer como disponíveis.
- A revelação ocorre em duas etapas: primeiro `?reveal=true` recarrega os valores, depois o componente `RevealValue` revela visualmente cada segredo.
