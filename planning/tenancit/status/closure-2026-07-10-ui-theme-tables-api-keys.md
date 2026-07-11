# Fechamento — preset da implementação de referência, tabelas e chaves de API

## Resultado

- Preset completo `b1FS9AzhY` aplicado (`base-nova`, olive/amber, Noto Sans e
  Lucide), com integrações específicas do Tenancit preservadas.
- Tabelas administrativas consolidadas nos primitives shadcn sem perda de
  busca, ordenação, paginação, responsividade ou interação por teclado.
- Criação de API client rejeita nomes duplicados ignorando caixa e espaços
  antes da chamada HTTP; token permanece one-shot, somente leitura e copiável.
- Ajustes de acessibilidade mantêm landmark único, navegação mobile, foco de
  diálogos e menus de locale/tema compatíveis com teclado.

## Evidências

- `bun run typecheck`, `bun run lint`: verdes.
- `bun run test`: 19 arquivos, 68 testes verdes.
- `make build`: verde; entry 70.712 B raw / 18.820 B gzip, seis rotas lazy.
- `make e2e` com retry zero: catálogo 15/15, 126/126 passos; 17 testes e smoke
  de rotas verdes.
- `diff -qr web/dist server/internal/spa/dist` e `git diff --check`: verdes.
- Vite/proxy validado no Browser em `http://localhost:5180`.

## Estado posterior

Este fechamento retrata o corte de UI anterior ao epic 02. No commit
`c5424d6`, scopes, expiração, RPM, rotação, rename, delete terminal, telemetria,
auditoria e limite global com Valkey já estão implementados. O estado canônico
da próxima iniciativa está no epic 03.
