# História 03 — Governança de sessões e settings

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Adaptar as novas superfícies do reference implementation para permitir inventário/revogação de
sessões e políticas runtime não secretas, com garantias mais fortes.

## Rastreabilidade

- Pesquisa reference implementation: sessões e settings.
- Nova tela `/security/sessions` e `/operations/settings`.

## Arquivos

- Registry tipado de settings, migration/queries e handlers transacionais.
- APIs/listagens de sessões; UI, i18n, auditoria e testes.

## Detalhe

Settings conhecidos incluem apenas políticas runtime, com versão/ETag,
validação, auditoria e update atômico. DSN, chaves AES, OIDC secret e credenciais
Valkey continuam fora do banco. Sessões podem ser revogadas conforme permissão;
a sessão atual sai por logout explícito.

## Tarefas

- [ ] Definir registry, defaults seguros e ownership de cada chave.
- [ ] Implementar compare-and-set/ETag e auditoria allowlisted.
- [ ] Listar sessões com last-used, expiry, principal e current.
- [ ] Revogar uma/todas por usuário e invalidar cache imediatamente.
- [ ] Criar telas responsivas e trilha E2E multilíngue.

## Verificação

Concorrência de settings, autorização por permissão, revogação multi-réplica,
ausência de secrets e testes Playwright.

