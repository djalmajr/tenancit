---
id: admin-to-consumer-golden-path
name: Configurar tenant no admin e consumir pelo caminho seguro identify → alias
reference: planning/tenancit/epics/01-tenancit/05-consumer-api-auth.md
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento com banco descartável ou namespace de teste único
  - token administrativo válido disponível: `tenancit_admin_dev`
  - é permitido usar um cliente HTTP externo no fim do fluxo para validar `/v1/resolve`
design_refs:
  overview: "planning/tenancit/proto/routes/overview.js"
  definitions-list: "planning/tenancit/proto/routes/definitions-list.js"
  tenants-list: "planning/tenancit/proto/routes/tenants-list.js"
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
  api-clients: "planning/tenancit/proto/routes/api-clients.js"
---

## Objetivo do usuário

Executar o caminho crítico completo: criar tipo de recurso, tenant, domínio,
resource e API client; identificar o tenant na borda sem segredos e resolver uma
configuração específica pelo hostname e alias estável sem cachear seus segredos.

## Passos (cada passo é uma AÇÃO de UI/API + o resultado esperado)

1. (`overview`) Autenticar-se se necessário → o painel administrativo fica acessível.
2. (`definitions-list`) Abrir **Recursos**, criar uma definition ativa com key única e adicionar campos `host` obrigatório e `password` obrigatório/segredo → a definition aparece com os campos corretos.
3. (`tenants-list`) Abrir **Tenants** e criar um tenant com slug único → a aplicação navega para o detalhe do tenant.
4. (`tenant-detail`) Abrir a aba **Domínios** e adicionar um hostname único → o hostname aparece na tabela do tenant.
5. (`tenant-detail`) Voltar para **Recursos**, adicionar recurso da definition criada, definir um alias único e preencher `host` e `password` → o recurso fica ativo na tabela e o segredo aparece mascarado no detalhe.
6. (`api-clients`) Abrir **Chaves de API**, conferir a orientação de `identify` e resolução por alias, criar uma chave e copiar o token exibido uma única vez → a chave aparece ativa na tabela.
7. (`consumer-api`) Em um cliente HTTP externo, chamar `GET /v1/identify?hostname=<hostname>` com `Authorization: Bearer <token>` → a resposta 200 contém somente `tenantSlug`, sem `resources` nem valores secretos.
8. (`consumer-api`) Chamar `GET /v1/resolve/<hostname>/resources/<alias>` com o mesmo token → a resposta 200 retorna somente o recurso solicitado e `Cache-Control: private, no-store`, sem `ETag`.
9. (`consumer-api`) Repetir `identify`/`resolve` sem o token → a resposta retorna 401.
10. (`consumer-api`) Chamar `identify` com hostname desconhecido e token válido → a resposta retorna 404.

## Resultado esperado

O caminho admin-to-consumer funciona de ponta a ponta sem entregar segredos à
borda: `identify` produz a identidade, o app resolve o alias necessário sem
cache compartilhado e recebe erros corretos para ausência de token, hostname ou
alias.

## Estado atual × design

- Este é um fluxo híbrido: usa UI para configurar dados e cliente HTTP externo para validar `/v1/identify` e `/v1/resolve?tenantId=`.
- A cobertura automatizada equivalente existe em `server/internal/httpapi/integration_test.go`, mas este fluxo valida a jornada operacional completa do painel até o contrato de consumo.
- A Consumer API não tem tela dedicada; se a persona exigir navegação puramente visual, este fluxo deve ser executado como validação assistida por terminal.
