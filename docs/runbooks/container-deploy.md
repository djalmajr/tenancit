# Deploy por imagem imutável

**Status:** AUTOMAÇÃO LOCAL VALIDADA em 2026-07-11; ativação em produção
aguarda a definição do primeiro alvo.

Este runbook descreve o contrato portável. Ele não incorpora hostnames, CIDRs,
secret manager, ingress ou unidades do reference implementation. Esses elementos precisam ser
registrados para o alvo real antes de declarar produção ativa.

## Invariantes

- imagem sempre referenciada por repositório + digest `sha256` de 64 hex;
- login de migration/owner separado dos logins runtime, jobs e backup;
- migration roda em comando one-shot antes de trocar as réplicas;
- runtime não importa o pacote de migration e não possui DDL;
- o binário anterior precisa continuar ready depois da fase expand;
- backup saudável e ainda fresh é pré-condição do rollout;
- duas réplicas compartilham PostgreSQL e Valkey; indisponibilidade do limiter
  falha fechada;
- rollback troca somente o digest, preservando DSN e schema;
- trusted forwarded headers permanecem desabilitados até existir uma allowlist
  derivada da topologia do ingress.

## Provisionar os logins PostgreSQL

O operador cria cinco logins no secret manager/PostgreSQL. O repositório não
gera nem persiste senhas. Em seguida, como owner administrativo:

```bash
export TENANCIT_POSTGRES_ADMIN_URL='postgres://...'
export TENANCIT_DATABASE_NAME='tenancit'
export TENANCIT_MIGRATION_LOGIN='tenancit_migration_prod'
export TENANCIT_RUNTIME_LOGIN='tenancit_runtime_prod'
export TENANCIT_JOBS_LOGIN='tenancit_jobs_prod'
export TENANCIT_BACKUP_LOGIN='tenancit_backup_prod'
export TENANCIT_REWRAP_LOGIN='tenancit_rewrap_prod'
export TENANCIT_JOBS_DATABASE_URL='postgres://tenancit_jobs_prod:...@db/tenancit?sslmode=require'
./deploy/postgres/configure-roles.sh
```

O comando transfere ownership do banco, schema e objetos públicos existentes ao
login de migration e concede membership nos grupos mínimos. Isso permite adotar
uma instalação que antes migrava como `postgres`; também pode ser executado
antes da primeira migration. Depois, `/migrate` reconcilia grants sobre objetos
atuais e default privileges.

## Configurar o rollout

Use `deploy/docker-compose.production.yml` como contrato de referência. Todas
as credenciais entram por ambiente injetado; o arquivo versionado não contém
defaults secretos. A rede de ingress já deve existir e oferecer TLS.

Variáveis adicionais do script:

- `TENANCIT_IMAGE` e `TENANCIT_IMAGE_DIGEST`;
- `TENANCIT_MIGRATION_DATABASE_URL` e `TENANCIT_RUNTIME_DATABASE_URL` distintos;
- `TENANCIT_PUBLIC_BASE_URL` HTTPS e `TENANCIT_NETWORK`;
- chave smoke `tenant:identify`, hostname/slug sentinela e credencial do
  reporter operacional;
- configuração OIDC, Valkey TLS/auth, AES e OTLP.

O preflight exige TLS também nos dois DSNs PostgreSQL (`sslmode=require`,
`verify-ca` ou `verify-full`) e `rediss://` no Valkey.

## Upgrade expand/contract

```bash
make deploy-preflight
./scripts/deploy-release.sh
```

O preflight verifica digest, render do Compose, ausência de DDL no runtime e
report de backup fresh. O release executa `/migrate`, consulta `/readyz` **antes**
de substituir as réplicas e aborta se o binário anterior não aceitar o schema
expandido. Só então sobe duas réplicas, aguarda readiness, executa o smoke e
publica report idempotente de migration.

O smoke de produção é não mutável: valida liveness/readiness, configuração e
redirect OIDC, fronteiras 401 e identifica um tenant sentinela com uma chave
dedicada de escopo mínimo. `post-deploy-smoke.sh`, que cria dados com bearer
administrativo, permanece restrito ao Compose local legado.

Migrações contract (remoção/NOT NULL/rename incompatível) não entram no mesmo
release que deixa de ler a forma antiga. Elas só podem ser aplicadas depois da
janela de observação e da confirmação de que nenhum digest anterior está ativo.

## Rollback

```bash
export TENANCIT_ROLLBACK_IMAGE_DIGEST='sha256:...'
./scripts/deploy-rollback.sh
```

O script rejeita tag mutável/digest inválido e troca apenas a imagem. Se uma
migration contract já tiver sido aplicada, rollback de binário deixa de ser
seguro; por isso contract é um release separado e explicitamente aprovado.

## Provas locais

```bash
make lint-deploy
make test-continuity
```

`test-continuity` sobe duas réplicas reais e prova bucket Valkey global,
revogação imediata observada pela outra réplica e atendimento após parar uma
instância. Os scripts de backup/restore publicam reports somente depois de
validar o artefato/restore.

## Gate do primeiro alvo

Antes de mudar este runbook para `VALIDADO em produção`, anexar ao handoff:

1. diagrama cliente -> ingress/TLS -> duas réplicas -> PostgreSQL/Valkey;
2. owners e origem das credenciais, sem seus valores;
3. allowlist de trusted proxies (ou confirmação de que continua desligada);
4. digest implantado, output sanitizado de preflight/smoke/readiness;
5. localização off-host do backup, restore drill e RPO/RTO medidos;
6. rollback real para o digest anterior e confirmação de dados preservados.
