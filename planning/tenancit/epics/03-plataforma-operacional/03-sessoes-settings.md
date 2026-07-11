# História 03 — Governança de sessões e settings

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Adaptar as novas superfícies da implementação de referência para permitir inventário/revogação de
sessões e políticas runtime não secretas, com garantias mais fortes.

## Rastreabilidade

- Pesquisa implementação de referência: sessões e settings.
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

- [x] Definir registry, defaults seguros e ownership de cada chave.
- [x] Implementar compare-and-set/ETag e auditoria allowlisted.
- [x] Listar sessões com last-used, expiry, principal e current.
- [x] Revogar uma/todas por usuário e invalidar cache imediatamente.
- [x] Criar telas responsivas e trilha E2E multilíngue.

## Verificação

Concorrência de settings, autorização por permissão, revogação multi-réplica,
ausência de secrets e testes Playwright.

## Evidência de fechamento

- Migration `00007` cria registry versionado e índices do inventário de sessões.
- `PATCH /v1/admin/settings` exige `If-Match`; revisão obsoleta recebe `412` e
  falha de auditoria reverte a alteração.
- A política de sessão é relida no create/authenticate; retenção de uso falha
  fechada quando o registry está indisponível.
- A sessão atual só termina por logout; outra sessão e todas as demais do mesmo
  principal podem ser revogadas transacionalmente e param de autenticar.
- `make e2e-oidc`: 2/2 cenários retry-zero com Dex, incluindo pt-BR/en-US,
  settings, duas sessões e revogação imediata.
