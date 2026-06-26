---
id: admin-to-consumer-golden-path
name: Configurar tenant no admin e resolver por hostname na Consumer API
reference: planning/konvario/epics/01-konvario/05-consumer-api-auth.md
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento com banco descartável ou namespace de teste único
  - token administrativo válido disponível: `konvario_admin_dev`
  - é permitido usar um cliente HTTP externo no fim do fluxo para validar `/v1/resolve`
design_refs:
  overview: "planning/konvario/proto/routes/overview.js"
  definitions-list: "planning/konvario/proto/routes/definitions-list.js"
  tenants-list: "planning/konvario/proto/routes/tenants-list.js"
  tenant-detail: "planning/konvario/proto/routes/tenant-detail.js"
  api-clients: "planning/konvario/proto/routes/api-clients.js"
---

## Objetivo do usuário

Executar o caminho crítico completo: criar tipo de recurso, tenant, domínio, resource e API client, então resolver o hostname como serviço consumidor.

## Passos (cada passo é uma AÇÃO de UI/API + o resultado esperado)

1. (`overview`) Autenticar-se se necessário → o painel administrativo fica acessível.
2. (`definitions-list`) Abrir **Recursos**, criar uma definition ativa com key única e adicionar campos `host` obrigatório e `password` obrigatório/segredo → a definition aparece com os campos corretos.
3. (`tenants-list`) Abrir **Tenants** e criar um tenant com slug único → a aplicação navega para o detalhe do tenant.
4. (`tenant-detail`) Abrir a aba **Domínios** e adicionar um hostname único → o hostname aparece na tabela do tenant.
5. (`tenant-detail`) Voltar para **Recursos**, adicionar recurso da definition criada e preencher `host` e `password` → o recurso fica ativo no tenant, o segredo aparece mascarado e **Prontidão para consumo** indica domínio/recurso/chave quando aplicável.
6. (`api-clients`) Abrir **Chaves de API**, conferir o snippet de `/v1/resolve`, criar uma chave e copiar o token exibido uma única vez → a chave aparece ativa na tabela.
7. (`consumer-api`) Em um cliente HTTP externo, chamar `GET /v1/resolve?hostname=<hostname>` com `Authorization: Bearer <token>` → a resposta retorna status 200, o slug do tenant e o recurso ativo.
8. (`consumer-api`) Repetir a chamada sem o token → a resposta retorna 401.
9. (`consumer-api`) Chamar com um hostname desconhecido e token válido → a resposta retorna 404.

## Resultado esperado

O caminho admin-to-consumer funciona de ponta a ponta: dados criados no painel são resolvidos pela Consumer API com API key ativa, segredo descriptografado sobre o contrato HTTP e erros corretos para ausência de token ou hostname desconhecido.

## Estado atual × design

- Este é um fluxo híbrido: usa UI para configurar dados e cliente HTTP externo para validar `/v1/resolve`.
- A cobertura automatizada equivalente existe em `server/internal/httpapi/integration_test.go`, mas este fluxo valida a jornada operacional completa do painel até o contrato de consumo.
- A Consumer API não tem tela dedicada; se a persona exigir navegação puramente visual, este fluxo deve ser executado como validação assistida por terminal.
