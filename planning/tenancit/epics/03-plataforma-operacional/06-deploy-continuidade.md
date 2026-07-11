# História 06 — Deploy, roles PostgreSQL e continuidade

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Transformar o plano genérico de deploy em automação portável e prova concreta
no primeiro alvo, sem incorporar topologia do reference implementation.

## Arquivos

- Container/Compose/manifests, migration command, scripts preflight/smoke.
- SQL/grants de runtime, migration, backup e jobs.
- Runbooks de deploy, TLS, secret rotation, restore, rollback e incidentes.

## Detalhe

Imagem imutável por digest, migration separada, runtime sem DDL, rollout
multi-réplica, trusted proxies explícitos, readiness gates, backup off-host,
restore drill e rollback preservando DSN/schema.

## Tarefas

- [ ] Criar perfis e grants mínimos por função PostgreSQL.
- [ ] Separar migrate do boot e testar compatibilidade expand/contract.
- [ ] Criar manifests/config genéricos e validação fail-loud.
- [ ] Automatizar preflight, smoke e rollback por digest.
- [ ] Integrar reports de backup/restore e alertas de freshness.
- [ ] Executar duas réplicas, failover, limiter e revogação imediata.
- [ ] Quando o alvo existir, registrar topologia e evidências reais.

## Verificação

Runtime não cria schema; migration role não atende HTTP; restore isolado passa
smoke; rollback não perde dados; produção só é marcada após evidência do alvo.

