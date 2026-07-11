# Migração de governança de API clients

**Status:** VALIDADO em PostgreSQL descartável.

O rollout é expand/contract. A migration `00003` expande o schema e preserva
clients existentes com ambos os scopes e política nula. A `00005` é o contract e
aborta se ainda houver `legacy_unbounded` ou client sem scope.

## Inventário e transição

1. Faça backup e confirme restore conforme `postgres-backup-restore.md`.
2. Aplique somente a fase expand durante a janela de compatibilidade.
3. Liste clients com `rpm_limit IS NULL OR expires_at IS NULL`, crie sucessores
   com menor privilégio e monitore `last_used_at`/uso diário.
4. Confirme tráfego no sucessor, revogue o anterior e repita até inventário zero.
5. Aplique o contract. Preview, RPM e expiração passam a ser `NOT NULL`, e a
   trigger rejeita `revoked -> active`.

## Rollback

Antes do contract, o binário anterior ainda lê o schema expandido. Depois do
contract, rollback exige restaurar um binário compatível com a política nova ou
executar explicitamente a migration Down; nunca ressuscite um token revogado.
O preflight falhar é um gate de segurança, não motivo para apagar dados.
