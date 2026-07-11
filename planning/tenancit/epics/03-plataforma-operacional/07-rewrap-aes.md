# História 07 — Rewrap AES executável

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Implementar integralmente o design existente como comando offline separado do
servidor HTTP.

## Arquivos

- `server/cmd/tenancit-rewrap`, serviço de campanha, queries e testes.
- Makefile/container job, métricas/reports e atualização do runbook/design.

## Detalhe

Advisory lock, inventário, dry-run completo, lotes `SKIP LOCKED`, decrypt/
encrypt/verify, CAS, retomada pelo estado do banco e zero alteração de ETag
funcional. Nenhum material criptográfico vai a argumento/log/telemetria.

## Tarefas

- [ ] Implementar preflight fail-closed e inventário por versão.
- [ ] Implementar dry-run que autentica todos os ciphertexts sem updates.
- [ ] Implementar lotes transacionais, CAS, cancelamento e no-progress timeout.
- [ ] Integrar métricas e system report sem IDs/material sensível.
- [ ] Cobrir writers concorrentes, crash, chave ausente e linha malformada.
- [ ] Ensaiar stage/cutover/rewrap/restore/retirada da chave antiga.

## Verificação

Property/integration tests, interrupção e rerun idempotente, restore
representativo e canários de vazamento.

