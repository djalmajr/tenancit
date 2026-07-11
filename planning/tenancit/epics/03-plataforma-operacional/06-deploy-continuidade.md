# História 06 — Deploy, roles PostgreSQL e continuidade

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Transformar o plano genérico de deploy em automação portável e prova concreta
no primeiro alvo, sem incorporar topologia do reference implementation.

## Responsabilidade, motivação e valor

Esta história não adiciona um novo domínio ao produto. Ela protege a operação
do próprio Tenancit, fonte crítica de configuração e secrets para vários
aplicativos. Se seu deploy, banco ou backup falhar, o dano pode atingir todos os
consumidores.

**Ganho:** menor privilégio entre HTTP, migration, jobs, backup e rewrap;
rollout/rollback previsíveis; restauração ensaiada; e continuidade
multi-réplica. O Tenancit não vira uma plataforma genérica de deploy nem
administra a infraestrutura dos recursos cadastrados.

## Arquivos

- Container/Compose/manifests, migration command, scripts preflight/smoke.
- SQL/grants de runtime, migration, backup e jobs.
- Runbooks de deploy, TLS, secret rotation, restore, rollback e incidentes.

## Detalhe

Imagem imutável por digest, migration separada, runtime sem DDL, rollout
multi-réplica, trusted proxies explícitos, readiness gates, backup off-host,
restore drill e rollback preservando DSN/schema.

## Tarefas

- [x] Criar perfis e grants mínimos por função PostgreSQL.
- [x] Separar migrate do boot e testar compatibilidade expand/contract.
- [x] Criar manifests/config genéricos e validação fail-loud.
- [x] Automatizar preflight, smoke e rollback por digest.
- [x] Integrar reports de backup/restore e alertas de freshness.
- [x] Executar duas réplicas, failover, limiter e revogação imediata.
- [ ] Quando o alvo existir, registrar topologia e evidências reais.

## Verificação

Runtime não cria schema; migration role não atende HTTP; restore isolado passa
smoke; rollback não perde dados; produção só é marcada após evidência do alvo.

## Evidência local — 2026-07-11

- `internal/migration` é importado pelo comando one-shot; `lint-go.sh` falha se
  `cmd/server` voltar a depender desse pacote.
- Login roles reais provaram migration owner, runtime sem DDL, backup read-only
  e jobs sem escrita de domínio; `make test-postgres-roles` também adota schema
  preexistente e roda na CI contra PostgreSQL real.
- `make test-continuity` alternou tráfego entre duas réplicas, comprovou bucket
  Valkey global, revogação observada pela outra instância e failover após stop.
- Dump PostgreSQL custom foi restaurado em banco isolado: 22 tabelas e o tenant
  sentinela foram preservados; reports de backup/restore ficaram healthy/fresh.
- Scripts rejeitam digest/tag inválidos, DSNs iguais e HTTP público. O release
  executa expand, exige que o digest anterior permaneça ready, sobe duas
  réplicas, roda smoke e reporta migration; rollback troca apenas o digest.

O item final permanece aberto por gate externo: não existe host/domínio/ingress,
secret manager ou política RPO/RTO fornecida. Isso não invalida a automação
genérica e não autoriza afirmar que produção está ativa.
