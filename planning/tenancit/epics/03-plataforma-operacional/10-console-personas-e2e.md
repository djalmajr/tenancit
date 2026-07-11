# História 10 — Console por persona e catálogo E2E

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Unificar as novas capacidades no padrão visual já assimilado do reference implementation,
sem expor ações que o principal não pode executar.

## Responsabilidade, motivação e valor

Administrador, operador, auditor e integrador usam o mesmo Tenancit, mas buscam
capacidades diferentes. A persona organiza navegação, linguagem e ações
permitidas; não cria quatro produtos nem substitui a autoridade RBAC do backend.

**Ganho:** menos ruído e erro humano, acessibilidade, três idiomas e prova dos
fluxos reais no navegador. O limite é uma única aplicação adaptativa, sem
dashboards paralelos ou personas hipotéticas sem uso.

## Arquivos

- App shell, rotas security/integrations/operations/activity e query keys.
- Data table persistida, i18n pt-BR/en-US/es-ES, flows/personas/scoreboard.

## Detalhe

Navegação agrupada por capacidade; UI deriva permissões do session endpoint;
backend continua autoridade. Preferências locais por tabela persistem colunas,
ordenação e page size e podem ser restauradas explicitamente. Filtros que no
futuro forem publicados na URL deverão prevalecer sobre o storage. Loading/
empty/error/stale e mobile/keyboard são obrigatórios.

## Tarefas

- [x] Criar IA e navegação por admin, operador, auditor e integrador.
- [x] Entregar sessões, settings, integrations, health e activity.
- [x] Persistir preferências e oferecer reset explícito.
- [x] Completar traduções e formatação locale-aware.
- [x] Automatizar fluxos OIDC, CSRF, revogação, webhook, DLQ, export e rewrap report.
- [x] Rodar scoreboard humano sem blockers/high pendentes.

## Verificação

Lint/type/unit/build, Playwright Vite e produto embutido, light/dark/mobile,
keyboard, três idiomas e matriz de permissões.

## Evidência entregue

- A navegação possui grupos Gestão, Operação e Segurança e mostra destinos
  somente quando a sessão declara a capacidade correspondente. Ações sensíveis
  nas telas também respeitam o mesmo conjunto; o backend permanece deny-by-default.
- A matriz de permissões persistida na sessão OIDC foi reconciliada com a matriz
  aplicada pelas rotas. O teste Dex comprovou settings, revogação de sessão,
  CSRF, logout e break-glass sem liberar capacidades por heurística visual.
- Tabelas persistem visibilidade, ordenação e tamanho de página por browser e
  superfície, preservam o formato legado e oferecem `Restaurar tabela`. O fluxo
  de diretórios prova persistência após reload e reset do padrão.
- Personas de administrador, operador, auditor e integrador têm objetivos e
  riscos explícitos; o scoreboard de 2026-07-11 encerrou com zero blocker/high.
- Gates: lint/typecheck, 21 arquivos/82 testes web e build/budget verdes;
  catálogo empacotado 22/22 + route smoke Vite; fluxo de preferências 2/2 e
  OIDC/Dex 2/2 sem retry.
