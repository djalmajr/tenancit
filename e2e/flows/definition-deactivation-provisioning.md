---
id: definition-deactivation-provisioning
name: Desativação de definition afeta o provisionamento no tenant
reference: docs/business/03-jornadas-operacionais.adoc#criar-um-novo-tipo-de-recurso
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - existe um tenant ativo para provisionar recursos
design_refs:
  definitions-list: "planning/tenancit/proto/routes/definitions-list.js"
  definition-detail: "planning/tenancit/proto/routes/definition-detail.js"
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
---

## Objetivo do usuário

Confirmar que o ciclo de vida (active/inactive) de uma resource definition controla, de forma coerente entre telas, se ela pode ou não ser provisionada num tenant.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Autenticar-se com `tenancit_admin_dev` → painel acessível.
2. (`definitions-list`) Criar (ou usar) uma definition ativa `toggle-e2e` com ao menos um campo obrigatório → a definition aparece como ativa.
3. (`tenant-detail`) Abrir o tenant de teste, clicar em **Adicionar recurso** e abrir o seletor de tipo → a definition `toggle-e2e` aparece entre os tipos disponíveis; fechar o diálogo sem salvar.
4. (`definition-detail`) Abrir `toggle-e2e` e clicar em **Desativar** → o status muda para inactive e a tela mostra feedback.
5. (`tenant-detail`) Voltar ao mesmo tenant, abrir **Adicionar recurso** novamente → a definition `toggle-e2e` **não** aparece mais entre os tipos; se ela era a única disponível, o diálogo mostra a mensagem de "nenhum tipo disponível / todos provisionados".
6. (`definition-detail`) Reabrir `toggle-e2e` e clicar em **Ativar** → o status volta para active com feedback.
7. (`tenant-detail`) Abrir **Adicionar recurso** outra vez → `toggle-e2e` volta a aparecer como tipo disponível.
8. (`overview`) Abrir a **Visão geral** após desativar/ativar → o KPI **Definições ativas** reflete a contagem correta em cada estado.

## Resultado esperado

Apenas definitions ativas são oferecidas para provisionamento; desativar remove a opção e reativar a restaura, de forma consistente entre o detalhe da definition, o diálogo de recurso do tenant e os KPIs da visão geral.

## Estado atual × design

- O filtro em `web/src/routes/tenant-detail.tsx` usa `definitions.filter(d => d.status === "active" && !activeKeys.has(d.key))`, então definitions inativas (ou já ativas no tenant) somem do seletor.
- A alternância de status vem de `PUT /v1/admin/resource-definitions/{id}/status`.
- Este fluxo valida o invariante cross-screen, complementando `resource-definition-management` (que cobre o CRUD da definition isoladamente).
