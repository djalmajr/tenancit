---
id: tenant-shared-resource-inheritance
name: Reutilizar configuração por vínculo, override e duplicação
reference: docs/developers/03-contratos-http.adoc#recursos-do-tenant
persona: administrator
entry: "http://localhost:5180/"
status: draft
preconditions:
  - app no ar e pessoa administradora autenticada
  - existem dois tenants e uma resource definition ativa com campos secretos e não secretos
design_refs:
  tenant-resources: "https://app.paper.design/file/01KMAVY7N0M4YTV51AD7DHA6Y8/01KMAVY7N14QH3DS7DMGQR9MZN"
---

## Objetivo do usuário

Reaproveitar uma configuração-base sem acoplar o domínio a “aplicações”, mantendo aliases estáveis, overrides locais explícitos e uma duplicação realmente independente.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`source-tenant`) Na entrada, clicar em **Tenants**, abrir o tenant de origem e selecionar **Recursos** → a tabela de recursos do tenant fica visível.
2. (`source-tenant`) Clicar em **Adicionar recurso**, escolher uma definition, informar um alias único e preencher os valores-base → a linha independente aparece com ícone de instância própria.
3. (`target-tenant`) Voltar à lista de tenants, abrir o tenant consumidor, selecionar **Recursos** e clicar em **Adicionar recurso** → o formulário oferece criar uma instância independente ou vinculá-la a uma origem compatível.
4. (`target-tenant`) Escolher o recurso-base, informar outro alias e salvar sem preencher todos os campos → a nova linha aparece com ícone de vínculo e herda os valores efetivos da origem.
5. (`linked-detail`) Abrir a linha vinculada e editar somente um campo → o modal identifica o override local e os demais campos continuam herdados.
6. (`source-detail`) Voltar ao recurso-base, alterar um campo que não possui override no vinculado e salvar → ao reabrir o recurso vinculado, o novo valor da origem aparece automaticamente.
7. (`linked-detail`) Usar **Voltar a usar origem** no campo com override → o valor local desaparece e o campo volta a acompanhar a origem.
8. (`linked-detail`) Clicar em **Duplicar**, informar um alias único e confirmar → nasce uma instância independente com snapshot dos valores efetivos e sem vínculo com a origem.
9. (`source-detail`) Alterar novamente o recurso-base → o vinculado acompanha a mudança, enquanto a duplicação preserva o snapshot anterior.
10. (`source-detail`) Tentar remover a origem enquanto existir um vínculo → a operação é bloqueada com orientação para desvincular, duplicar ou remover os dependentes primeiro.

## Resultado esperado

O operador distingue visualmente recursos independentes e vinculados, entende herança e overrides, cria cópias independentes por duplicação e não consegue quebrar dependências silenciosamente.
