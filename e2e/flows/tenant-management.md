---
id: tenant-management
name: Cadastrar e manter tenant com domínio
reference: docs/business/03-jornadas-operacionais.adoc#cadastrar-um-tenant
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - usar valores únicos por rodada, por exemplo slug `acme-e2e` e hostname `app.acme-e2e.local`
design_refs:
  tenants-list: "planning/tenancit/proto/routes/tenants-list.js"
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
---

## Objetivo do usuário

Cadastrar um cliente, associar um hostname exato, validar sua manutenção e
encerrar a rodada removendo-o com confirmação forte.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`tenants-list`) Autenticar-se se o diálogo de token aparecer e clicar em **Tenants** na sidebar → a lista de tenants aparece com busca e botão **"Novo tenant"**.
2. (`tenants-list`) Clicar em **Novo tenant** → o diálogo **"Novo tenant"** abre com campos Nome e Slug.
3. (`tenants-list`) Preencher Nome e Slug e clicar em **Criar tenant** → o diálogo fecha e a aplicação navega para o detalhe do tenant criado.
4. (`tenant-detail`) Clicar em **Editar**, alterar nome ou status e clicar em **Salvar** → o cabeçalho do tenant reflete os dados persistidos e a tela mostra feedback de atualização.
5. (`tenant-detail`) Revisar **Prontidão para consumo** e clicar na aba **Domínios** → a tabela ou estado vazio de domínios fica visível.
6. (`tenant-detail`) Clicar em **Adicionar**, preencher Hostname e clicar em **Adicionar** → o hostname aparece na tabela de domínios e a prontidão passa a indicar domínio presente.
7. (`tenants-list`) Voltar para **Tenants** pela breadcrumb ou sidebar e buscar pelo nome ou slug criado → a lista filtra para o tenant esperado.
8. (`tenant-detail`) Voltar ao detalhe e clicar na lixeira do domínio criado → o diálogo de confirmação informa o hostname; confirmar **Remover** remove o domínio, mostra feedback e o estado vazio informa que ao menos um domínio é necessário para resolver o tenant.
9. (`tenant-detail`) Na **Zona de perigo**, clicar em **Excluir tenant permanentemente**, conferir a descrição da cascata e tentar confirmar sem preencher o slug → o botão permanece desabilitado.
10. (`tenant-detail`) Digitar exatamente o slug do tenant e confirmar → o tenant, seus domínios/resources/valores são removidos, a UI volta para **Tenants** e o registro não aparece mais na busca.

## Resultado esperado

O tenant pode ser criado, editado, encontrado por busca, receber/remover domínio
e ser excluído permanentemente somente após confirmação pelo slug.

## Estado atual × design

- A lista e o detalhe existem em `web/src/routes/tenants.tsx` e `web/src/routes/tenant-detail.tsx`.
- A UI atual cria tenant sem domínio no mesmo modal; domínio é adicionado em seguida no detalhe.
- Remoção de domínio usa confirmação dedicada antes da ação destrutiva.
- Desativação continua sendo a opção reversível; hard-delete é uma ação separada e destrutiva.
