# Fechamento: principal administrativo e autorização

**Fonte:** `.agents/plans/admin-principal-authorization-foundation.md`
**Modo:** Closure por story

## Contexto

- **Iniciativa:** fundação de identidade e autorização administrativa.
- **Período:** 2026-07-10.
- **Objetivo:** introduzir principal não secreto e autorização deny-by-default,
  sem alterar o login por token ou os contratos HTTP e da SPA.

## Resultado

- O token administrativo validado produz o principal técnico
  `shared_admin_token/primary`, sem bearer ou hash no contexto.
- As permissões fechadas `admin.read`, `tenant.write`, `resource.write`,
  `secret.reveal`, `tenant.hard_delete` e `api_client.manage` são aplicadas às
  22 rotas administrativas.
- Principal ausente falha com `401`; permissão ausente falha com `403`, ambos
  antes do handler.
- `GET .../resources?reveal=true` exige `admin.read` e `secret.reveal`; a
  listagem mascarada exige somente `admin.read`.
- O token compartilhado preserva todas as permissões atuais, mantendo a
  compatibilidade até OIDC/RBAC.

## Plano versus resultado

- **Entregue:** todos os critérios de aceite e tarefas do plano.
- **Pendente nesta story:** nada.
- **Fora de escopo e ainda pendente:** OIDC, sessão, CSRF, RBAC configurável,
  audit log, migrations e decisões de IdP/topologia/retenção.
- **Desvio de escopo:** nenhum criado nesta story. O endurecimento Bearer em
  `auth.go`/`auth_test.go` e `SecurityHeaders` em `server.go` já existiam no
  worktree e foram preservados sem alteração funcional nesta entrega.

## Verificação executada

| Verificação | Resultado |
|---|---|
| RED/GREEN | compilação inicialmente falhou pela fundação ausente; testes direcionados verdes após implementação |
| Mutação | `hasPermission` sempre permissivo fez o teste falhar com `204` em vez de `403`; restauração verde |
| Lint Go | `make lint-go`, verde |
| Web lint/typecheck/unit | `make test-web`, 19 arquivos e 67 testes verdes |
| Go com PostgreSQL | `REQUIRE_DB_TESTS=1 go test -count=1 ./...`, verde |
| Build/embed | `make build`, budgets verdes e `web/dist` idêntico ao embed |
| Catálogo E2E | 15/15 flows e 126/126 passos válidos |
| Playwright | 17/17 testes mais route smoke 1/1, retry zero |
| Higiene | `git diff --check` limpo; nenhum container E2E residual |
| Browser | Vite/proxy em `:5180` exibiu dashboard autenticado após restart da API |
| Documentação | plano e registro de encerramento atualizados |

## Riscos remanescentes

- O ator ainda é técnico e compartilhado; não há atribuição individual até
  OIDC/sessão.
- A fundação autoriza, mas ainda não persiste eventos de auditoria.
- O token compartilhado continua com autoridade total e deve tornar-se
  break-glass quando a identidade humana for entregue.

## Próximos passos

- [ ] Planejar a primitiva transacional append-only de audit log usando o
      principal já disponível no contexto.
- [ ] Resolver IdP, issuer/audience, claims/grupos, roles, origin e política de
      sessão antes da story de OIDC.

## Verificação do status

- [x] O status reflete o estado executado.
- [x] Riscos e dependências externas possuem próximo passo explícito.
- [x] O próximo passo é observável.

## Próximo passo recomendado

Abrir a próxima `/agile-story` para a primitiva transacional do audit log.
