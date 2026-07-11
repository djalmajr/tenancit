# História 01 — Schema expandível e contratos

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Expandir persistência e tipos sem alterar o acesso dos clients existentes.

## Rastreabilidade

Designs de API clients e auditoria; ADR 0004.

## Arquivos

- `server/migrations`: migration expand.
- `server/internal/store/queries`: queries e sqlc.
- `server/internal/httpapi`: views e contratos compatíveis.

## Detalhe

Adicionar campos nullable, scopes retrocompatíveis, uso diário e auditoria
append-only. Clients antigos recebem ambos os scopes e continuam ilimitados até
a retirada coordenada.

## Tarefas

- [x] RED de schema, compatibilidade e ausência de hash.
- [x] Migration expand e geração sqlc.
- [ ] Views com status efetivo e marcador legado.
- [ ] Atualizar documentação do modelo e rollout.

## Verificação

`make test-db`, `make test`, `make build`, migration up/down e canário de hash.
