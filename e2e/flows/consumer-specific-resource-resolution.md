---
id: consumer-specific-resource-resolution
name: Resolver recurso específico por hostname e definition key
reference: docs/engenharia/03-contratos-http.adoc#consumer-api
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento com banco descartável ou namespace de teste único
  - token administrativo válido disponível: `konvario_admin_dev`
  - é permitido usar um cliente HTTP externo no fim do fluxo para validar `/v1/resolve/{hostname}/resources/{definitionKey}`
design_refs:
  definitions-list: "planning/konvario/proto/routes/definitions-list.js"
  tenants-list: "planning/konvario/proto/routes/tenants-list.js"
  tenant-detail: "planning/konvario/proto/routes/tenant-detail.js"
  api-clients: "planning/konvario/proto/routes/api-clients.js"
---

## Objetivo do usuário

Configurar um tenant com dois recursos e validar que um serviço consumidor consegue buscar somente o recurso solicitado pela `definitionKey`.

## Passos (cada passo é uma AÇÃO de UI/API + o resultado esperado)

1. (`overview`) Autenticar-se se necessário → o painel administrativo fica acessível.
2. (`definitions-list`) Abrir **Recursos** e criar duas definitions ativas com keys únicas, por exemplo `postgres-e2e` e `vault-e2e`, cada uma com ao menos um campo obrigatório → as definitions aparecem com campos suficientes para provisionamento.
3. (`tenants-list`) Abrir **Tenants** e criar um tenant com slug único → a aplicação navega para o detalhe do tenant.
4. (`tenant-detail`) Abrir a aba **Domínios** e adicionar um hostname único → o hostname aparece na tabela do tenant.
5. (`tenant-detail`) Voltar para **Recursos**, adicionar um recurso para cada definition criada e preencher seus campos obrigatórios → os dois recursos aparecem ativos no tenant.
6. (`api-clients`) Abrir **Chaves de API**, criar uma chave e copiar o token exibido uma única vez → a chave aparece ativa na tabela.
7. (`consumer-api`) Em um cliente HTTP externo, chamar `GET /v1/resolve/<hostname>/resources/<definitionKey-1>` com `Authorization: Bearer <token>` → a resposta retorna status 200 somente com o recurso da definition solicitada.
8. (`consumer-api`) Chamar `GET /v1/resolve/<hostname>/resources/<definitionKey-inexistente>` com token válido → a resposta retorna 404 com erro de resource não encontrado.
9. (`consumer-api`) Revogar a chave pela UI em **Chaves de API** e repetir a chamada válida → a resposta retorna 401.

## Resultado esperado

A Consumer API permite recuperar um recurso específico por hostname e definition key, distingue resource inexistente de autenticação inválida e respeita revogação de API client.

## Estado atual × design

- Este é um fluxo híbrido: a UI prepara dados e o cliente HTTP valida o endpoint server-to-server.
- Não há tela dedicada para `/v1/resolve/{hostname}/resources/{definitionKey}`; se o endpoint não for citado na UI, isso pode virar achado de orientação para integradores.
