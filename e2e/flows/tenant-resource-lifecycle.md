---
id: tenant-resource-lifecycle
name: Provisionar, inspecionar e manter um recurso independente
reference: docs/business/03-jornadas-operacionais.adoc#revelar-segredos-no-painel
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - existe um tenant ativo
  - existe uma resource definition ativa com ao menos um campo obrigatório e um campo segredo
design_refs:
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
  definition-detail: "planning/tenancit/proto/routes/definition-detail.js"
---

## Objetivo do usuário

Provisionar uma instância identificada por alias, inspecionar e editar seus campos com segurança e controlar seu ciclo de vida.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`tenant-detail`) Autenticar-se se necessário e clicar em **Tenants** na sidebar → a lista de tenants aparece.
2. (`tenant-detail`) Clicar no tenant de teste e selecionar a aba **Recursos** → a tabela mostra alias, tipo, contagens, status, busca, ordenação e colunas.
3. (`tenant-detail`) Clicar em **Adicionar recurso** → o diálogo permite escolher qualquer definition ativa, inclusive uma já usada pelo tenant.
4. (`tenant-detail`) Selecionar a definition e informar um alias único válido → os campos dinâmicos aparecem e o alias explica como a instância será resolvida.
5. (`tenant-detail`) Preencher os campos obrigatórios, incluindo o segredo, e clicar em **Salvar recurso** → o diálogo fecha e uma nova linha independente aparece com ícone próprio, sem impedir outra instância do mesmo tipo.
6. (`tenant-detail`) Clicar na linha criada → um modal exibe alias, tipo, status e tabela de campos; o segredo permanece mascarado.
7. (`tenant-detail`) Clicar no ícone de editar de um campo não secreto, salvar um novo valor e reabrir o recurso → o valor atualizado aparece sem alterar o alias.
8. (`tenant-detail`) Clicar em **Revelar** no rodapé → todos os valores secretos do recurso ficam visíveis somente nessa sessão do modal; clicar em **Ocultar** volta a mascará-los.
9. (`tenant-detail`) Clicar em **Desativar** no modal → o status da linha muda para inativo e a ação passa a reativar.
10. (`tenant-detail`) Reabrir o recurso e clicar em **Reativar** → o status volta para ativo e a tela mostra feedback.
11. (`tenant-detail`) Clicar em **Remover**, revisar a confirmação e confirmar → somente a instância escolhida sai da tabela, sem remover outras do mesmo tipo.

## Resultado esperado

O recurso usa alias único, respeita campos dinâmicos, mascara segredos por padrão e pode ser inspecionado, editado, desativado, reativado e removido sem confundir instâncias do mesmo tipo.

## Estado atual × design

- O fluxo está implementado em `web/src/routes/tenant-detail.tsx`.
- O mesmo tipo pode ser instanciado várias vezes; a constraint é o alias normalizado único dentro do tenant.
- A tabela abre um modal por linha. O botão de rodapé alterna revelação/ocultação dos valores secretos e cada campo editável tem ação explícita.
