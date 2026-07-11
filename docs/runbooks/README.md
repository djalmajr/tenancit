# Runbooks operacionais

Este índice separa procedimentos comprovados de planos ainda dependentes de
decisões de ambiente. Um runbook só recebe status **VALIDADO** depois de ser
executado de ponta a ponta no ambiente indicado.

| Runbook | Status | Ambiente / última validação |
| --- | --- | --- |
| [Bootstrap local](local-bootstrap.md) | VALIDADO | Docker Desktop local, 2026-07-09 |
| [Smoke pós-deploy](post-deploy-smoke.md) | VALIDADO | Compose dev local, 2026-07-09 |
| [Backup e restore PostgreSQL](postgres-backup-restore.md) | VALIDADO | Compose local, 2026-07-09 |
| [Valkey e rate limit](valkey-rate-limit.md) | VALIDADO | Compose local e teste multi-instância, 2026-07-10 |
| [Migração de API clients](api-client-governance-migration.md) | VALIDADO | PostgreSQL descartável, schema v5, 2026-07-10 |
| [Auditoria e retenção](admin-audit-and-retention.md) | VALIDADO | PostgreSQL descartável, 2026-07-10 |
| [Deploy em host](container-deploy.md) | PLANO | aguarda definição do primeiro host/alvo |
| [Rewrap de chave AES](aes-key-rewrap.md) | RASCUNHO | aguarda implementação e ensaio do job |

Regras:

- comandos devem ser reexecutáveis ou declarar claramente o efeito destrutivo;
- todo procedimento de mudança inclui pré-condição, verificação e rollback;
- tokens, chaves AES e dumps ficam fora do repositório;
- `docker compose down` preserva dados; `make docker-reset
  CONFIRM=destroy-local-data` é o único atalho documentado que remove volumes.
