# História 03 — Scopes e expiração

**Origin:** `planning/tenancit/epics/02-governanca-operacional/00-overview.md`

## Contexto

Aplicar menor privilégio e validade absoluta sem interromper legado.

## Rastreabilidade

`docs/developers/design/api-client-policy.md`.

## Arquivos

Middleware Consumer API, handlers admin, store e formulário de API clients.

## Detalhe

Scopes fechados `tenant:identify` e `resource:resolve`; create exige política
completa; desconhecido/revogado/expirado compartilham 401 e falta de scope usa
403.

## Tarefas

- [ ] RED da matriz rota/scope e boundary de relógio.
- [ ] Principal do client e enforcement.
- [ ] Create/edit e UI com presets de validade.
- [ ] Compatibilidade `legacy_unbounded`.

## Verificação

Unit, integração HTTP, E2E de scopes e regressão dos clients antigos.
