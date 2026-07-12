# História 05 — Validar o primeiro ambiente real

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: fixtures locais comprovam contratos, mas não comprovam IdP, proxies,
TLS, secrets, continuidade ou recuperação numa topologia Kubernetes. O objetivo
desta história é executar os runbooks no laboratório pessoal e registrar
evidência sanitizada. O ganho é validar integração e recuperação sem confundir
esse ensaio com homologação de produção de um cliente.

## Rastreabilidade

- `docs/runbooks/container-deploy.md`.
- `docs/runbooks/post-deploy-smoke.md`.
- `docs/runbooks/aes-key-rewrap.md`.
- ADRs 0005–0007 e designs de auditoria/API clients.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| Runbooks em `docs/runbooks/` | Atualizar | Trocar premissas genéricas por evidência sanitizada do alvo |
| Configuração de deploy escolhida | Adicionar em caminho próprio sem secrets | Codificar topologia reproduzível |
| Registro de validação e roadmap | Atualizar | Distinguir validado de apenas implementado |
| História/overview | Atualizar | Persistir gates externos e resultados |
| `docs/runbooks/first-production-validation-record.md` | Preencher | Centralizar evidência sanitizada e aceite |

## Detalhe

AS-IS: duas réplicas, Valkey, Dex, backup/restore e rewrap passam localmente.
TO-BE: o laboratório valida issuer/audience/claims, ingress/TLS, secrets,
health/readiness, retenção funcional, failover, restore e ensaio de rewrap.
Cada ambiente de cliente mantém um gate próprio para IdP corporativo, HA,
backup off-site, observabilidade e valores aprovados de SLO/RPO/RTO.

### Aceite

- IdP real autentica e aplica roles/claims esperados; break-glass é testado e
  permanece excepcional/auditado.
- TLS, origins, cookies, CSP e trusted proxies passam testes negativos.
- Duas réplicas provam limiter global, revogação imediata e failover.
- Backup/restore é medido, o smoke passa sobre o restore e os limites do
  laboratório são registrados sem prometer RPO/RTO de produção.
- Rewrap executa dry-run e ensaio no restore antes de qualquer campanha real.
- Retenção e owners do laboratório são documentados; SLO, RPO, RTO e alertas
  permanecem gates obrigatórios por ambiente de cliente.

### Dependências

História 03 e disponibilidade de ambiente/IdP/políticas. Nenhuma decisão é
necessária agora; esses itens permanecem gates explícitos até existirem.

## Tarefas

- [x] Criar o registro sanitizado, matriz de gates e entrada mínima necessária.
- [x] Registrar alvo, owners e matriz de dependências sem credenciais.
- [x] Configurar IdP/OIDC e validar claims/RBAC/CSRF/logout.
- [x] Validar ingress, TLS, DNS, secret manager e trusted proxies.
- [x] Executar deploy por digest, migration, smoke e rollback.
- [x] Executar falhas controladas de Valkey, Postgres e uma réplica.
- [x] Medir backup/restore e registrar limites de RPO/RTO do laboratório.
- [x] Executar dry-run de rewrap.
- [x] Registrar limites de SLO, retenção, alertas e evidências sanitizadas.

## Verificação

Usar os comandos canônicos dos runbooks, incluindo `make smoke`,
`make test-continuity`, restore verificado e rewrap dry-run. Nenhuma saída
versionada pode conter DSN, token, cookie, chave ou payload secreto.

## Evidência

Descoberta em 2026-07-11:

- configurações locais/versionadas foram pesquisadas sem abrir arquivos de
  credenciais;
- foi encontrado somente um manifesto de laboratório legado, com uma réplica,
  imagem local, migration no boot e configuração de secrets incompatível com
  os contratos atuais; ele não é um alvo representativo aceitável;
- o único contexto Kubernetes configurado não estava alcançável e nenhum outro
  domínio, manifesto ou contexto Tenancit ativo foi encontrado.

Em 2026-07-11 o cluster pessoal K3s/Hetzner foi validado como primeiro alvo real
de teste. O chart portátil e o perfil pessoal estão em `deploy/helm/tenancit`.
OIDC/TLS, migrations, duas réplicas, limiter global, falhas de Valkey/PostgreSQL,
backup/restore e rewrap passaram conforme o registro sanitizado.

O aceite desta história não promove o laboratório a produção: ambientes de
clientes continuam responsáveis por IdP corporativo, HA multi-node, backup
off-site, observabilidade e SLO/RPO/RTO aprovados.
