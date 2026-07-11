# História 10 — Console por persona e catálogo E2E

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Unificar as novas capacidades no padrão visual já assimilado do reference implementation,
sem expor ações que o principal não pode executar.

## Arquivos

- App shell, rotas security/integrations/operations/activity e query keys.
- Data table persistida, i18n pt-BR/en-US/es-ES, flows/personas/scoreboard.

## Detalhe

Navegação agrupada por capacidade; UI deriva permissões do session endpoint;
backend continua autoridade. Preferências por tabela persistem colunas,
ordenação e page size; URL vence storage. Loading/empty/error/stale e mobile/
keyboard são obrigatórios.

## Tarefas

- [ ] Criar IA e navegação por admin, operador, auditor e integrador.
- [ ] Entregar sessões, settings, integrations, health e activity.
- [ ] Persistir preferências e oferecer reset explícito.
- [ ] Completar traduções e formatação locale-aware.
- [ ] Automatizar fluxos OIDC, CSRF, revogação, webhook, DLQ, export e rewrap report.
- [ ] Rodar scoreboard humano sem blockers/high pendentes.

## Verificação

Lint/type/unit/build, Playwright Vite e produto embutido, light/dark/mobile,
keyboard, três idiomas e matriz de permissões.

