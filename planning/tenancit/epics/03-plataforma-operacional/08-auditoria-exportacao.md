# História 08 — Auditoria, retenção e exportação

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Completar a operação da trilha já transacional com identidade humana, saúde de
partições, retenção segura e exportação controlada.

## Responsabilidade, motivação e valor

O Tenancit administra configurações, credenciais e revelações de secrets; por
isso deve explicar quem fez cada ação sensível e preservar evidência durante um
incidente. Partições e retenção impedem crescimento sem controle; legal hold
impede descarte durante investigação.

**Ganho:** rastreabilidade, investigação e exportação limitada para segurança.
O produto não implementa um SIEM, análise forense completa ou storage WORM
próprio; integrações externas ficam atrás de interface opcional.

## Arquivos

- Queries/APIs de activity, export jobs e retenção/partition maintenance.
- Tela consolidada de atividade, downloads one-shot e adapters opcionais.
- Runbooks de retenção, legal hold, SIEM/WORM e incidentes.

## Detalhe

Export exige `audit.export`, janela/limite, filtro allowlisted, arquivo cifrado
ou streaming autenticado e auditoria do próprio acesso. Eventos sobrevivem a
hard delete. Retenção não remove legal hold e é observável.

## Tarefas

- [x] Propagar OIDC e eventos de login/logout/CSRF/RBAC à trilha.
- [x] Expor saúde/futuras partições e job idempotente de retenção.
- [x] Implementar export pequeno síncrono e grande assíncrono com expiração.
- [x] Consolidar UI de activity com filtros/paginação server-side.
- [x] Definir interface opcional para SIEM/WORM sem vendor obrigatório.

## Verificação

Append-only/permissions, janela máxima, legal hold, export expiry, auditoria do
download e canários de dados proibidos.

## Evidência entregue

- Migration `00010` adiciona registry, legal holds e jobs; a função
  `SECURITY DEFINER` é a única fronteira capaz de drenar a partição default,
  criar/dropar partições e respeita holds antes da retenção.
- `/tenancit-audit-jobs` usa o login `tenancit_jobs`; runtime não recebe DDL nem
  `EXECUTE` na função privilegiada. O teste de roles prova as duas fronteiras.
- Export de até 1.000 eventos termina síncrono; acima disso usa claim
  `SKIP LOCKED`. Janela máxima é 31 dias, teto absoluto 100.000, payload fica
  cifrado e expira em 24 h. Download destrói o ciphertext e grava o próprio
  acesso na mesma transação antes de devolver bytes. `Idempotency-Key` fica
  vinculado ao fingerprint de filtros/formato e rejeita reuso divergente.
- Consultas da activity também gravam `audit.events_read`; filtros e cursores
  são aplicados no servidor. UI exibe saúde, export e legal holds em pt/en/es.
- `ArchiveSink` é a interface opcional de SIEM/WORM, síncrona e fail-closed,
  sem SDK ou fornecedor embutido.
- Gates locais: Go estrito com PostgreSQL, separação dos roles, lint/typecheck,
  build com quatro binários, 21 E2E empacotados + route smoke Vite e 2 E2E OIDC
  passaram sem retry.
