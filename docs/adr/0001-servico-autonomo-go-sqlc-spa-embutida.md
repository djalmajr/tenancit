# ADR 0001 — Serviço autônomo em Go, sqlc e SPA embutida

- **Status:** Aceito
- **Data:** 2026-06-23

## Contexto

O Tenancit atende o hot path de configuração multi-tenant:
serviços consumidores precisam resolver recursos por hostname com baixa
complexidade operacional. A solução precisa ser pequena, previsível e fácil de
rodar junto de PostgreSQL.

As alternativas consideradas foram:

- incorporar a configuração em cada aplicação consumidora;
- criar um BFF ou painel separado para administração;
- usar ORM completo para o domínio;
- manter API e frontend em serviços distintos.

Essas alternativas aumentariam acoplamento ou superfície operacional para um
serviço cujo domínio é pequeno e centrado em SQL.

## Decisão

Implementar o serviço como um binário Go autônomo:

- router HTTP com `chi`;
- acesso a dados com `pgx` + `sqlc`;
- migrations com `goose`;
- SPA admin React buildada separadamente e embutida via `embed.FS`;
- PostgreSQL como única dependência persistente.

## Consequências

Positivas:

- deploy simples: um binário/container;
- SQL explícito e type-safe;
- sem CORS entre painel e API;
- menor footprint no caminho de consumo.

Custos:

- build final precisa compilar web e server;
- mudanças em contratos HTTP exigem atenção entre Go e TypeScript;
- painel admin fica acoplado ao release do serviço.

## Status

Aceito em 2026-06-23. Fonte de verdade de implementação: `server/`, `web/`,
`Dockerfile` e `Makefile`.
