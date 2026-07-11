# Handoff — Tenancit

- **Snapshot:** 2026-07-11
- **Base observada:** `301b470` mais contract final validado nesta sessão
- **Entrega Git anterior:** console por persona está publicado em `668625f`;
  o gate de escala foi corrigido e publicado até `301b470`, verde na CI
  (`29168210762`). O contract final aguarda o commit desta sessão.
- **Backlog da rodada:** os 25 itens originais estão `DONE` em
  [`plans/README.md`](../plans/README.md)

Este arquivo registra o estado executável da rodada. Decisões duráveis continuam
nos ADRs e designs; contratos normativos continuam em `docs/developers/`.

## Resultado atual

- Consumer API rejeita tenants inativos, usa erros públicos estáveis, limita
  bodies e não permite cache HTTP de payloads com secrets.
- Hostnames são canonicalizados; ETags cobrem identidade, resources, fields e
  values; `identify` exige revalidação a cada uso para não manter a identidade
  anterior após reatribuição; resolução e admin montam fields/values em batch,
  sem N+1.
- Mutações aninhadas validam o parent, `key_hash` não sai da API e field em uso
  retorna conflito estável.
- Leituras administrativas com `reveal=true` usam `Cache-Control: private,
  no-store` no servidor e `cache: "no-store"` no cliente, reduzindo a exposição
  de cleartext em caches do navegador.
- A Admin API usa Authorization Code + PKCE no backend, sessão opaca hash-only,
  cookie `HttpOnly`, CSRF/origin e RBAC deny-by-default. OIDC audita `iss` +
  `sub`; o bearer só permanece como modo legado dev ou break-glass opt-in,
  versionado, hash-only e read-only.
- O console lista sessões administrativas, distingue a atual e revoga outras
  imediatamente. Settings runtime não secretos usam registry fechado, revisão
  global/ETag, compare-and-set e auditoria na mesma transação.
- Outbox transacional publica change feed `events:read` e webhooks HMAC com
  targets cifrados, SSRF defense, lease, retry, circuito, DLQ/replay e retenção.
- OTLP/HTTP configurável exporta traces e métricas RED/USE com dimensões
  allowlisted. `/readyz` prova PostgreSQL/Valkey/IdP sem expor configuração; o
  console reúne probes, filas, circuitos e reports append-only com freshness.
- Toda mutação administrativa e reveal bem-sucedidos escrevem auditoria
  append-only na mesma transação. Negativas/erros também são registrados sem
  headers, body ou query string; a consulta usa filtros exatos e cursor keyset.
- Login/logout, CSRF/RBAC, leitura da activity, legal hold e download de export
  também entram na trilha. Export CSV/JSONL é limitado, cifrado, expira em 24 h
  e é consumido uma vez; `tenancit-audit-jobs` drena a default, mantém partições
  futuras e aplica retenção sem atravessar holds.
- Create tenant, provision resource e create/rotate API client exigem UUID
  idempotente escopado por principal/operação. Retry idêntico reproduz a mesma
  resposta cifrada; payload divergente falha com conflito e nenhum novo efeito.
- API clients exigem scopes fechados, RPM e expiração; suportam edição, rotação
  one-shot, revogação terminal e hard delete apenas após revogar. Uso diário e
  `last_used_at` alimentam o console operacional sem armazenar tokens/hashes.
- A migration de contract tornou preview/RPM/expiração/scopes obrigatórios. O
  marcador transitório `legacy_unbounded` saiu da API e da SPA; o filtro e os
  fallbacks legados não fazem parte do contrato final.
- Rate limit global usa Valkey com operação atômica e relógio do servidor,
  retorna `429` com headers contratuais e falha fechado com `503` quando o
  limiter está indisponível. O modo em memória é somente para dev explícito.
- SPA mascara secrets sem manter cleartext escondido, aplica timeout uniforme,
  separa inativação de hard delete e ensina o fluxo
  `identify -> tenantId -> ETag`.
- Todas as telas administrativas usam TanStack Query com keys e invalidações
  centralizadas. Reveal e criação one-shot ficam fora dos caches; troca de
  credencial cancela requests, desmonta observers e apaga dados protegidos.
- O console agrupa Gestão, Operação e Segurança e filtra navegação e ações pelas
  permissões da sessão. Preferências de tabela persistem colunas, ordenação e
  page size por superfície e possuem reset explícito; o backend continua sendo
  a autoridade RBAC.
- As seis páginas de negócio carregam por rota lazy; budgets automáticos
  protegem o entry e impedem chunks acima de 500 kB.
- O catálogo Playwright automatiza 20/20 flows e 147/147 passos em PostgreSQL
  efêmero, cobrindo produto empacotado e Vite/proxy com cleanup verificado.
  Um E2E vertical adicional entrega evento a um receiver real, recalcula HMAC e
  remove seu tenant ao terminar; outro gate usa Dex real para login/logout,
  sessão, CSRF e break-glass.
- A imagem final aplica CSP/HSTS/nosniff/frame-deny/referrer/permissions headers.
- Migrations saíram do boot HTTP: `/migrate` possui owner/DDL e `/server` não
  importa o pacote de migration. Grupos PostgreSQL separam runtime, jobs e
  backup; deploy genérico usa digest, preflight, expand/contract e rollback sem
  trocar DSN/schema.
- `/tenancit-rewrap` executa rotação AES offline com inventário/dry-run integral,
  advisory lock, lotes `SKIP LOCKED`, CAS, retomada, OTLP e report dedicado. O
  role PostgreSQL só atualiza cipher/nonce/version; chaves nunca são argumentos.
- Compose preserva PostgreSQL em volume nomeado; reset destrutivo exige
  confirmação explícita; smoke, bootstrap e backup/restore locais têm runbooks.

## Runtime local deixado ativo

| Superfície | URL/porta | Papel |
|---|---|---|
| Vite + HMR | `http://localhost:5180/` | painel de desenvolvimento; proxy para `api-dev` |
| API dev | `http://localhost:8081/` | Go via `go run` |
| Produto empacotado | `http://localhost:8080/` | SPA embutida + Go |
| PostgreSQL | `localhost:5432` | volume `tenancit-postgres-data` |
| Valkey | `localhost:6379` | buckets globais de rate limit |

O token e a chave do Compose são somente de demonstração. Não os promover para
outro ambiente.

## Gates e evidências

| Gate | Último resultado nesta rodada |
|---|---|
| Web lint | ESLint, zero warnings |
| Web typecheck | `tsc --noEmit`, exit 0 |
| Web unit | 21 arquivos / 82 testes, todos verdes |
| Go estrito | `REQUIRE_DB_TESTS=1 go test -count=1 ./...`, incluindo testcontainers, verde |
| Produto | Docker multi-stage com lockfile frozen, SPA + binário Go, verde |
| Bundle/embed | seis rotas lazy; entry abaixo do budget; `make build` sincroniza `web/dist` e o embed Go |
| HTTP empacotado | `/` e `/healthz` 200; headers defensivos presentes; `reveal=true` com `private, no-store` |
| Smoke | health, 401s, create, identify, resolve, ETag/304 e cleanup, verde |
| Catálogo E2E | três stacks novas: 22/22 testes empacotados + route smoke Vite em cada run, retry-zero; OIDC/Dex 2/2 |
| Escala | duas rodadas limpas em 100/500/1.000/5.000; `KEEP_FULL_LISTS`; primeiro checkpoint em 500 definições reais |
| Browser | Vite `:5180` autenticado; saúde mostra PostgreSQL/Valkey e reports de backup/restore saudáveis |
| Persistência | tenant sentinela sobreviveu a `down/up` sem remover volume |
| Backup/restore | dump custom PostgreSQL 16, checksum obrigatório, 22 tabelas e tenant preservados; reports healthy |
| Continuidade | duas réplicas: limiter global, revogação cross-replica e failover, verde |
| Rewrap | CLI completo, clone restaurado, concorrência/falhas/retomada e retirada da chave antiga, verde |
| Documentação | `asciidoctor -o /dev/null docs/README.adoc`, verde |

Comandos canônicos:

```bash
make test-web
make test-db
make build
make e2e
make e2e-oidc
make e2e-stability
make benchmark-scale
docker compose config
make smoke
make lint-deploy
make test-continuity
TENANCIT_ROLE_TEST_ADMIN_URL='postgres://...' make test-postgres-roles
```

O smoke exige credenciais locais válidas conforme
[`post-deploy-smoke.md`](runbooks/post-deploy-smoke.md). Ele cria dados com
prefixo único e executa cleanup; não use tokens de produção em logs ou tickets.

## Decisões atuais e propostas

| Tema | Fonte | Estado |
|---|---|---|
| Produto autônomo, secrets e fronteiras atuais | [ADRs 0001–0004](adr/README.md) | Aceito/implementado |
| Principal administrativo e autorização por rota | [Plano](../.agents/plans/admin-principal-authorization-foundation.md) | Implementado para o token compartilhado |
| Identidade humana admin | [ADR 0005](adr/0005-identidade-humana-admin-oidc-sessoes-rbac.md) | Implementada e validada com Dex; ativação real depende do IdP |
| Auditoria admin append-only | [Design](developers/design/admin-audit-log.md) | Principal legado, OIDC e break-glass implementados |
| Política de API clients | [Design](developers/design/api-client-policy.md) / [ADR 0006](adr/0006-valkey-rate-limit-global.md) | Implementada; contract schema v5 |
| Rewrap AES | [Design](developers/design/aes-key-rewrap.md) / [runbook](runbooks/aes-key-rewrap.md) / [plano 022](../plans/022-aes-key-rewrap-spike.md) | Implementado e validado localmente; campanha real depende do alvo |
| Trajetória e dependências | [Roadmap](business/04-escopo-e-roadmap.adoc) | Atualizado |

## Decisões externas ainda necessárias

1. **Ativação de identidade:** IdP, issuer/audience, claims/grupos, roles e origin de produção.
2. **Topologia:** alvo de deploy, ingress/TLS, DNS, secret manager e trusted
   proxies. Duas réplicas estão provadas localmente; CIDR permanece desligado.
3. **Auditoria:** retenção organizacional e materialização dos logins PostgreSQL
   já separados por contrato no alvo de produção.
4. **Rewrap:** aprovadores, tamanho de lote e repetição do ensaio no restore do
   alvo antes de qualquer rotação real.

Essas dependências impedem afirmar que o deploy/IdP real ou a rotação AES estão
ativados. O contrato OIDC, auditoria, governança de clients e rate limit global
estão implementados e validados localmente; ainda exigem configuração do alvo.

## Próxima sequência recomendada

O plano persistente e decomposto para essa sequência está em
[`epic 03`](../planning/tenancit/epics/03-plataforma-operacional/00-overview.md),
baseado também na análise das novidades do reference implementation em 2026-07-11.

1. Publicar o contract final e confirmar a CI remota.
2. Coletar cardinalidade real; reabrir paginação ao observar/projetar 500
   registros em uma superfície, conforme o relatório de escala.
3. Escolher o primeiro alvo e repetir o rewrap no restore real, além de validar o
   [`container-deploy.md`](runbooks/container-deploy.md) com TLS, secrets,
   backup/restore e smoke.

## Limites conscientes desta rodada

- Paginação server-side não foi implementada: a curva sintética encontrou o
  primeiro breakpoint confirmado em 500 definições, mas não há volume
  operacional declarado que justifique mudar o contrato agora.
- O benchmark deve ser repetido com `TENANCIT_SCALE_OBSERVED_VOLUME` igual à
  cardinalidade real ou prevista antes de abrir o epic de paginação.
- A automação de deploy está validada localmente, mas o runbook não afirma
  produção ativa até existir um alvo. Não foram copiadas credenciais, CIDRs,
  host networking ou automações específicas do reference implementation.
- A fixture Dex prova identidade humana; a atribuição real depende de ativar um
  IdP corporativo com mappings revisados. O modo legado continua explicitamente
  técnico e nunca é apresentado como pessoa.
