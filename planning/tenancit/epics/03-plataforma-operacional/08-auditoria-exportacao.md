# História 08 — Auditoria, retenção e exportação

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Completar a operação da trilha já transacional com identidade humana, saúde de
partições, retenção segura e exportação controlada.

## Arquivos

- Queries/APIs de activity, export jobs e retenção/partition maintenance.
- Tela consolidada de atividade, downloads one-shot e adapters opcionais.
- Runbooks de retenção, legal hold, SIEM/WORM e incidentes.

## Detalhe

Export exige `audit.export`, janela/limite, filtro allowlisted, arquivo cifrado
ou streaming autenticado e auditoria do próprio acesso. Eventos sobrevivem a
hard delete. Retenção não remove legal hold e é observável.

## Tarefas

- [ ] Propagar OIDC e eventos de login/logout/CSRF/RBAC à trilha.
- [ ] Expor saúde/futuras partições e job idempotente de retenção.
- [ ] Implementar export pequeno síncrono e grande assíncrono com expiração.
- [ ] Consolidar UI de activity com filtros/paginação server-side.
- [ ] Definir interface opcional para SIEM/WORM sem vendor obrigatório.

## Verificação

Append-only/permissions, janela máxima, legal hold, export expiry, auditoria do
download e canários de dados proibidos.

