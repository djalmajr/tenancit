# Handoff — Tenancit

- **Snapshot:** 2026-07-11
- **Base observada:** `e3a1da5` mais a fatia OIDC/Dex em validação desta sessão
- **Entrega Git anterior:** `e3a1da5` está em `main`; push da fatia corrente vem após os gates
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
- Toda mutação administrativa e reveal bem-sucedidos escrevem auditoria
  append-only na mesma transação. Negativas/erros também são registrados sem
  headers, body ou query string; a consulta usa filtros exatos e cursor keyset.
- API clients exigem scopes fechados, RPM e expiração; suportam edição, rotação
  one-shot, revogação terminal e hard delete apenas após revogar. Uso diário e
  `last_used_at` alimentam o console operacional sem armazenar tokens/hashes.
- Rate limit global usa Valkey com operação atômica e relógio do servidor,
  retorna `429` com headers contratuais e falha fechado com `503` quando o
  limiter está indisponível. O modo em memória é somente para dev explícito.
- SPA mascara secrets sem manter cleartext escondido, aplica timeout uniforme,
  separa inativação de hard delete e ensina o fluxo
  `identify -> tenantId -> ETag`.
- Todas as telas administrativas usam TanStack Query com keys e invalidações
  centralizadas. Reveal e criação one-shot ficam fora dos caches; troca de
  credencial cancela requests, desmonta observers e apaga dados protegidos.
- As seis páginas de negócio carregam por rota lazy; budgets automáticos
  protegem o entry e impedem chunks acima de 500 kB.
- O catálogo Playwright automatiza 18/18 flows e 139/139 passos em PostgreSQL
  efêmero, cobrindo produto empacotado e Vite/proxy com cleanup verificado.
  Um gate adicional usa Dex real para login/logout, sessão, CSRF e break-glass.
- A imagem final aplica CSP/HSTS/nosniff/frame-deny/referrer/permissions headers.
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
| Web unit | 19 arquivos / 73 testes, todos verdes |
| Go estrito | `REQUIRE_DB_TESTS=1 go test -count=1 ./...`, incluindo testcontainers, verde |
| Produto | Docker multi-stage com lockfile frozen, SPA + binário Go, verde |
| Bundle/embed | seis rotas lazy; entry abaixo do budget; `make build` sincroniza `web/dist` e o embed Go |
| HTTP empacotado | `/` e `/healthz` 200; headers defensivos presentes; `reveal=true` com `private, no-store` |
| Smoke | health, 401s, create, identify, resolve, ETag/304 e cleanup, verde |
| Catálogo E2E | 18/18 testes + route smoke Vite, retry-zero; OIDC/Dex 1/1 retry-zero |
| Escala | duas rodadas em 100/500/1.000/5.000; `KEEP_FULL_LISTS`; checkpoint em 1.000 registros reais |
| Browser | Vite `:5180` validado em login, dashboard, API clients, token one-shot, snippets e hard delete; console final sem erros |
| Persistência | tenant sentinela sobreviveu a `down/up` sem remover volume |
| Backup/restore | dump custom restaurado em banco isolado e conferido |
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
| Rewrap AES | [Design](developers/design/aes-key-rewrap.md) / [runbook](runbooks/aes-key-rewrap.md) / [plano 022](../plans/022-aes-key-rewrap-spike.md) | Spike concluído; job não implementado |
| Trajetória e dependências | [Roadmap](business/04-escopo-e-roadmap.adoc) | Atualizado |

## Decisões externas ainda necessárias

1. **Ativação de identidade:** IdP, issuer/audience, claims/grupos, roles e origin de produção.
2. **Topologia:** alvo de deploy, ingress/TLS, DNS, secret manager, quantidade de
   réplicas e trusted proxies. CIDR não deve ser habilitado antes desse desenho.
3. **Auditoria:** retenção organizacional e separação dos roles PostgreSQL de
   runtime/migration no alvo de produção.
5. **Rewrap:** implementação do CLI/job, tamanho de lote e ensaio integral em um
   restore representativo antes de qualquer rotação real.

Essas dependências impedem afirmar que o deploy/IdP real ou a rotação AES estão
ativados. O contrato OIDC, auditoria, governança de clients e rate limit global
estão implementados e validados localmente; ainda exigem configuração do alvo.

## Próxima sequência recomendada

O plano persistente e decomposto para essa sequência está em
[`epic 03`](../planning/tenancit/epics/03-plataforma-operacional/00-overview.md),
baseado também na análise das novidades do reference implementation em 2026-07-11.

1. Estabilizar a CI remota e manter a documentação reconciliada com `main`.
2. Entregar governança de sessões/settings e fluxos de revogação administrativa.
3. Separar roles PostgreSQL de runtime/migration e definir retenção/partições no
   primeiro alvo.
4. Escolher o primeiro alvo e validar o
   [`container-deploy.md`](runbooks/container-deploy.md) com TLS, secrets,
   backup/restore e smoke.
5. Implementar outbox/webhooks, observabilidade e idempotência administrativa.
6. Implementar e ensaiar o job de rewrap; só depois retirar chaves históricas.

## Limites conscientes desta rodada

- Paginação server-side não foi implementada: a curva sintética encontrou o
  primeiro breakpoint confirmado em 1.000 registros, mas não há volume
  operacional declarado que justifique mudar o contrato agora.
- O benchmark deve ser repetido com `TENANCIT_SCALE_OBSERVED_VOLUME` igual à
  cardinalidade real ou prevista antes de abrir o epic de paginação.
- O runbook de deploy continua `PLANO` até existir um alvo. Não foram copiadas
  credenciais, CIDRs, host networking ou automações específicas do reference implementation.
- A fixture Dex prova identidade humana; a atribuição real depende de ativar um
  IdP corporativo com mappings revisados. O modo legado continua explicitamente
  técnico e nunca é apresentado como pessoa.
