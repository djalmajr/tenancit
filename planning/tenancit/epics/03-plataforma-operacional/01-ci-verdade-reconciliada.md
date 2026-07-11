# História 01 — CI verde e verdade reconciliada

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

O pipeline abre um PostgreSQL por teste e executa packages em paralelo, causando
resets no GitHub Actions. Além disso, epic 02, handoff e roadmap contradizem o
commit atual. A história entrega uma base confiável para todo o restante.

## Rastreabilidade

- `.github/workflows/ci.yml`, `server/internal/testsupport/postgres.go`.
- `planning/tenancit/epics/02-governanca-operacional` e `docs/HANDOFF.md`.

## Arquivos

- Modificar workflow, testsupport e Makefile para tiers unit/integration.
- Atualizar epic 02, handoff, roadmap, ADR/design indexes e status closures.
- Adicionar diagnóstico/artifacts de banco em falha.

## Detalhe

TO-BE: testes rápidos independentes; integração usa PostgreSQL service único por
job com isolamento por database/schema, ou pool estritamente limitado. Cleanup
é determinístico e `REQUIRE_DB_TESTS=1` permanece fail-closed.

Aceite: três runs CI verdes sem retry; race/leak não aparece; documentos não
descrevem capability implementada como pendente.

## Tarefas

- [x] Reproduzir e registrar a falha remota e o pico de containers/conexões.
- [ ] Separar unitários de integração sem reduzir cobertura.
- [x] Implementar isolamento por database/schema e cleanup seguro.
- [ ] Adicionar timeout e logs úteis sem DSN/credencial.
- [ ] Executar unit, integração, race aplicável, build e E2E crítico.
- [x] Marcar epic 02 entregue e reconciliar documentos canônicos.

## Verificação

`make lint-go`, `make test-web`, `make test-db`, `make build`, `make e2e-pr` e
três execuções do workflow principal.
