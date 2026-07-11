# Política de API clients

- **Status:** Implementado (expand/contract concluído no schema v5)
- **Data:** 2026-07-09
- **Escopo:** credenciais server-to-server da Consumer API
- **Relacionados:** [ADR 0004](../../adr/0004-admin-token-e-api-clients.md),
  [contrato HTTP da Consumer API](../03-contratos-http.adoc) e
  [design de auditoria administrativa](./admin-audit-log.md)

## Objetivo

Evoluir os API clients de uma credencial apenas ativa/revogada para uma
identidade operacional com menor privilégio, expiração, limite de uso e sinais
que permitam responder com segurança:

- o que esse client pode chamar;
- até quando o token é válido;
- qual volume ele pode produzir;
- quando foi usado pela última vez;
- se está saudável, ocioso, expirado, limitado ou sendo abusado.

O desenho preserva a decisão do ADR 0004: o token bruto é exibido uma única
vez, somente seu hash é persistido e API clients não autenticam a superfície
administrativa.

## Estado atual

Hoje `api_clients` guarda `id`, `name`, `key_hash`, `status` e `created_at`.
Um token ativo autentica igualmente:

- `GET /v1/identify`;
- `GET /v1/resolve` por hostname ou `tenantId`;
- `GET /v1/resolve/{hostname}/resources/{definitionKey}`.

Não há escopo, expiração, limite de requisições, `last_used_at` nem histórico
agregado de uso. A listagem admin já omite `key_hash` e a criação já devolve o
token bruto apenas uma vez; esses invariantes devem permanecer.

## Princípios

1. **Menor privilégio por capacidade.** Um client de edge que só identifica um
   tenant não recebe permissão para resolver segredos.
2. **Falhas de credencial não viram oráculo.** Token desconhecido, revogado ou
   expirado devolve a mesma resposta pública `401`.
3. **Sem ilimitado por acidente.** Novos clients exigem expiração e RPM
   positivos explícitos. Exceções precisam ser deliberadas, visíveis e
   auditadas.
4. **Revogação continua imediata.** A primeira versão não adiciona cache local
   de autenticação que possa manter um token revogado válido em outra réplica.
5. **Limite de segurança é global.** Em múltiplas réplicas, todos os processos
   consomem o mesmo bucket distribuído.
6. **Observabilidade não vaza credenciais.** Token, hash, header
   `Authorization` e query string bruta nunca entram em logs, métricas, uso ou
   auditoria.
7. **Métrica operacional não é faturamento.** Contadores podem ser agregados em
   lote; autenticação, escopo, expiração e rate limit são síncronos e não podem
   depender desse lote.

## Modelo de autorização

### Catálogo inicial de scopes

| Scope | Rotas | Dados acessíveis |
|---|---|---|
| `tenant:identify` | `/v1/identify` | apenas `tenantSlug`; nunca resources ou secrets |
| `resource:resolve` | `/v1/resolve` e `/v1/resolve/{hostname}/resources/{definitionKey}` | configuração do tenant, inclusive fields secretos descriptografados |

Os scopes são independentes. `resource:resolve` não concede implicitamente
`tenant:identify`, e vice-versa. A criação exige pelo menos um scope conhecido;
scope vazio ou desconhecido recebe `400` com código estável. A lista é fechada
e versionada no servidor, não aceita strings arbitrárias.

O middleware deve autenticar uma única vez e colocar no contexto um principal
verificado com `api_client_id`, `name` e scopes. Handlers não recebem token nem
hash. O pipeline da Consumer API fica:

1. extrair somente `Authorization: Bearer <token>`;
2. calcular o hash e buscar client + scopes em uma query;
3. rejeitar desconhecido, revogado ou expirado com `401 invalid_api_key`;
4. rejeitar scope ausente com `403 insufficient_scope`;
5. consultar o rate limiter compartilhado;
6. executar o handler e registrar sinais operacionais sem segredo.

O suporte alternativo a `X-API-Key` observado no reference implementation não deve ser copiado:
o contrato atual do Tenancit já usa Bearer e ampliar formas de entrada aumenta a
superfície sem necessidade demonstrada.

Scopes v1 autorizam **capacidade**, não restringem tenants específicos. Caso o
produto precise delegar apenas um subconjunto de tenants, isso requer uma
política própria (`api_client_tenants` ou regra equivalente), sem codificar
tenant IDs dentro do nome do scope.

### Status e expiração

O estado persistido continua `active|revoked`. O estado efetivo apresentado
pela API e pelo painel é calculado nesta ordem:

1. `revoked`, quando `status = 'revoked'`;
2. `expired`, quando `status = 'active'` e `now() >= expires_at`;
3. `active`, nos demais casos.

`expires_at` é absoluto, em UTC, e não desliza com o uso. Para novos clients:

- a API exige `expires_at` futuro;
- o painel sugere 90 dias e oferece presets de 30, 90, 180 e 365 dias;
- o máximo inicial é 365 dias;
- não existe opção silenciosa “nunca expira”.

Clients legados podem permanecer temporariamente com `expires_at = NULL`, mas
são marcados como `legacy_unbounded` e precisam ser substituídos durante a
migração. Não se atribui uma data retroativa automaticamente, pois isso poderia
derrubar consumidores sem coordenação.

`revoked` deve ser terminal no modelo alvo. Se a operação precisar de uma pausa
reversível, deve-se introduzir um estado explícito `suspended`; reativar um token
marcado como revogado contradiz a resposta a comprometimento. O endpoint atual
que aceita `revoked -> active` permanece somente durante a janela de
compatibilidade e depois responde `409 client_must_rotate`.

## Rate limit por client

### Contrato

`rpm_limit` representa requisições autenticadas por minuto e é aplicado por
`api_client_id`, nunca por IP. Novos clients exigem inteiro positivo explícito;
`0` e `NULL` não significam ilimitado fora da compatibilidade legada.

O algoritmo recomendado é token bucket, com reposição contínua e capacidade
derivada do RPM. O check ocorre depois de autenticação e scope, antes do handler.
Ao exceder:

- status `429 Too Many Requests`;
- erro estável `rate_limited`;
- `Retry-After` e os headers contratuais `RateLimit-Limit`,
  `RateLimit-Remaining` e `RateLimit-Reset`;
- incremento de métrica e agregado de uso, sem gerar uma linha de auditoria por
  request.

O valor inicial sugerido pelo painel não deve ser inventado em código. Ele deve
vir de configuração operacional calibrada com tráfego real; a API continua
exigindo que o operador confirme um valor positivo no create.

### Múltiplas réplicas

Um mapa em memória por processo **não é solução definitiva**. Duas réplicas com
limite 60 RPM e buckets locais permitiriam até 120 RPM, além de perder o estado
em restart e distribuir tráfego de forma desigual.

O servidor deve depender de uma interface `RateLimiter` com operação atômica.
Em produção com mais de uma réplica, a implementação obrigatória é compartilhada
e distribuída — por exemplo Valkey/Redis com script/function atômica, ou outro
serviço que prove as mesmas garantias. A chave do bucket usa o ID estável do
client, nunca o token ou seu hash.

Requisitos do backend:

- decisão atômica entre réplicas concorrentes;
- expiração automática de buckets inativos;
- relógio consistente para refill/reset;
- timeout curto e métricas de latência/erro;
- teste de integração que alterna requests entre pelo menos duas instâncias.

Se o limiter compartilhado estiver indisponível, a Consumer API falha fechada
com `503 rate_limiter_unavailable`. Fazer fallback silencioso para buckets
locais quebraria o limite justamente durante uma falha. Uma implementação
in-memory pode existir apenas no modo local, com uma réplica, habilitação
explícita e aviso no boot; ela não satisfaz o gate de produção.

## Persistência proposta

### `api_clients`

Adicionar, sem remover os campos existentes:

| Coluna | Tipo | Regra |
|---|---|---|
| `token_preview` | `text` | prefixo não secreto capturado na criação; nunca derivado do hash |
| `rpm_limit` | `integer` | positivo para novos clients; `NULL` apenas legado durante migração |
| `expires_at` | `timestamptz` | obrigatório para novos clients; `NULL` apenas legado |
| `last_used_at` | `timestamptz` | último request que autenticou com sucesso |
| `revoked_at` | `timestamptz` | preenchido na transição para revogado |
| `updated_at` | `timestamptz` | mudança de nome/política/status |

Scopes ficam em `api_client_scopes(api_client_id, scope)`, com PK composta,
foreign key `ON DELETE CASCADE` e check para o catálogo conhecido. A consulta do
middleware agrega os scopes em uma única ida ao PostgreSQL.

`last_used_at` significa “a credencial autenticou”, inclusive quando o handler
depois responde `4xx` por input ou tenant desconhecido. Token inválido,
revogado, expirado, sem scope ou bloqueado antes da autenticação não atualiza o
campo. Escritas são coalescidas por client e persistidas no máximo uma vez por
minuto com `GREATEST(last_used_at, novo_valor)`, de forma que réplicas não façam
o relógio andar para trás. Um atraso de até um minuto é aceitável para inventário,
mas deve estar documentado na UI.

### `api_client_usage_daily`

Agregado operacional proposto:

| Coluna | Regra |
|---|---|
| `day` | data UTC |
| `api_client_id` | UUID lógico, sem FK para preservar histórico após delete |
| `operation` | `identify` ou `resolve`; não armazena URL/query bruta |
| `status_class` | `2`, `3`, `4` ou `5` |
| `request_count` | requests autenticados que chegaram ao handler |
| `rate_limited_count` | recusas `429` do client/scope válido |

PK: `(day, api_client_id, operation, status_class)`. Cada réplica acumula deltas
curtos e faz `INSERT ... ON CONFLICT DO UPDATE` somando contadores. A perda
máxima aceita em crash deve ser declarada e testada; esses dados não podem ser
usados para cobrança. Retenção inicial: seis meses completos em UTC, com job
explícito, métrica de sucesso/falha e teste do cutoff.

## Observabilidade e auditoria

### Logs estruturados

Uma request autenticada registra somente:

- `request_id`;
- `api_client_id`;
- nome da operação (`identify`/`resolve`), método e route template;
- status, duração e resultado do rate limit;
- opcionalmente tamanho de resposta, sem body.

Não registrar nome mutável como identidade primária, hostname/tenant da query,
token, preview, hash, header, ETag, body ou valor de resource. Erros de banco
seguem os códigos públicos e a política de log sanitizado já adotada no
Tenancit.

### Métricas

Métricas mínimas:

- `consumer_requests_total{operation,status_class}`;
- `consumer_rate_limited_total{operation}`;
- `consumer_auth_failures_total{reason}` com razões agregadas que não formem
  oráculo no HTTP;
- latência por operação;
- latência/erros do limiter compartilhado;
- atraso e falhas do flush de `last_used_at`/usage.

Evitar `client_name`, hostname, tenant ou request ID como labels. Se for
necessário diagnóstico por client, usar consulta admin ao agregado no banco; não
introduzir cardinalidade ilimitada no sistema de métricas.

### Auditoria administrativa

O design `admin-audit-log.md` registra o ciclo de vida, não cada request de
consumo:

- `api_client.created`;
- `api_client.policy_updated` com before/after allowlisted de scopes, RPM e
  expiração;
- `api_client.revoked`;
- `api_client.rotated`;
- `api_client.deleted`.

Nenhum evento contém token, `key_hash` ou preview. A criação bem-sucedida grava
client + evento na mesma transação antes de devolver o token uma única vez. A
trilha de uso diário e os logs respondem por tráfego; a auditoria responde por
mudança administrativa.

## Contrato admin proposto

Todos os endpoints continuam sob a autenticação administrativa; autorização
humana/RBAC é tratada pelo design próprio de identidade admin.

| Método e path | Contrato alvo |
|---|---|
| `POST /v1/admin/api-clients` | exige `Idempotency-Key`, recebe `name`, `scopes`, `rpm_limit`, `expires_at`; devolve `{client, token}` e permite replay do mesmo envelope cifrado por 10 minutos |
| `GET /v1/admin/api-clients` | lista política e sinais operacionais, nunca hash/token |
| `PATCH /v1/admin/api-clients/{id}` | altera nome, scopes, RPM ou expiração futura de client ainda ativo; exige pelo menos um scope e não ressuscita expirado/revogado |
| `POST /v1/admin/api-clients/{id}/revoke` | revoga imediatamente e de forma terminal |
| `POST /v1/admin/api-clients/{id}/rotate` | exige `Idempotency-Key`, cria um sucessor e devolve o mesmo token em retries idênticos por 10 minutos; revoga o anterior conforme grace period explícito |
| `GET /v1/admin/api-clients/{id}/usage?from=&to=` | uso agregado por dia/operação/status, com intervalo máximo e paginação/limite |
| `DELETE /v1/admin/api-clients/{id}` | hard delete somente se já revogado; uso e auditoria permanecem |

Resposta de list/detail:

```json
{
  "id": "uuid",
  "name": "edge-prod",
  "key_preview": "tnc_abcd…",
  "status": "active",
  "scopes": ["tenant:identify"],
  "rpm_limit": 300,
  "created_at": "2026-07-09T12:00:00Z",
  "updated_at": "2026-07-09T12:00:00Z",
  "expires_at": "2026-10-07T12:00:00Z",
  "last_used_at": null,
  "legacy_unbounded": false
}
```

O exemplo preserva o estilo `snake_case` já usado pelo contrato de API clients;
tipos internos da SPA podem mapear esses nomes sem alterar o payload HTTP.
Erros estáveis: `invalid_scope`, `invalid_expiration`, `invalid_rpm`,
`client_must_rotate` e `rate_limiter_unavailable`.

## Admin UI

### Criação

O modal de criação passa a exigir:

- nome;
- scopes com descrição clara de que `resource:resolve` pode devolver secrets;
- expiração, com 90 dias pré-selecionados e data final visível;
- RPM positivo confirmado pelo operador.

Depois de criar, o token permanece em modal one-time com copiar e confirmação
de que foi armazenado. Fechar o modal descarta o valor; reload/listagem nunca o
recupera.

### Inventário e operação

A tabela mostra nome, preview, scopes, status efetivo, RPM, expiração,
`last_used_at` e uso recente. Filtros mínimos: status, scope, “expira em 30 dias”,
“nunca usado” e “legado sem limite/expiração”. Datas usam UTC no contrato e o
fuso do browser apenas para apresentação.

Ações:

- editar política, com resumo before/after;
- revogar em confirmação destrutiva e imediata;
- rotacionar em fluxo que mostra o novo token uma vez e deixa explícito quando
  o anterior será revogado;
- consultar uso;
- hard delete somente em client já revogado e em área de perigo separada.

O painel alvo não oferece “reativar” para uma credencial revogada. Clients
expirados orientam rotação, não extensão retroativa do mesmo token. Linhas
`legacy_unbounded` recebem alerta até serem substituídas.

## Migração e compatibilidade

### Fase 1 — schema compatível

1. Adicionar novas colunas nullable e `api_client_scopes`.
2. Preencher **ambos** os scopes em todos os clients atuais, preservando exatamente
   o acesso que já possuem.
3. Manter `rpm_limit = NULL` e `expires_at = NULL` para legados, interpretados
   temporariamente como ilimitado/sem expiração.
4. Adicionar checks como `NOT VALID`, corrigir inventário e só então validar.
5. Implantar código que lê o schema novo antes de começar a exigir os campos em
   writes novos.

Nenhum client existente é revogado ou recebe prazo retroativo nessa fase.

### Fase 2 — política para novos clients

- create exige scopes, RPM e expiração;
- middleware passa a aplicar scope/expiry;
- limiter distribuído deve estar saudável antes de ativar RPM em produção;
- lista/UI destaca legados e oferece rotação mantendo a política escolhida;
- monitorar `401`, `403`, `429`, `503` e limiter por uma janela de rollout.

### Fase 3 — retirada do legado

1. Rotacionar consumidores legados e validar uso do sucessor via
   `last_used_at`/contadores.
2. Revogar o anterior; manter rollback somente pelo token sucessor já entregue,
   não por reativação do comprometido.
3. Quando não houver `legacy_unbounded`, tornar `rpm_limit` e `expires_at`
   obrigatórios no banco.
4. Retirar `revoked -> active` do endpoint e da UI.

Esta fase foi materializada pela migration `00005_api_client_governance_contract.sql`.
Ela possui preflight que bloqueia o contract enquanto houver client legado ou
sem scopes, torna preview/RPM/expiração obrigatórios e impede reativação também
no banco. Em bases existentes, execute o inventário e a rotação antes de aplicar
essa migration; não há expiração retroativa automática.

Deploy exige ordem expand/contract. Rollback do binário durante a fase expand
continua possível porque os campos antigos não mudam e os novos aceitam `NULL`.
O contract final só ocorre depois de encerrar a janela de rollback.

## Critérios de aceite

### Autenticação e scopes

- [ ] Client legado continua acessando identify e resolve imediatamente após a
      migração expand.
- [ ] Client só com `tenant:identify` recebe `200/304` em identify e `403` em
      todas as variantes de resolve.
- [ ] Client só com `resource:resolve` recebe `403` em identify e pode resolver.
- [ ] Scope vazio/desconhecido é rejeitado na criação/edição.
- [ ] Desconhecido, revogado e expirado retornam o mesmo `401 invalid_api_key`.
- [ ] Nenhuma resposta admin contém `key_hash`; token bruto só aparece em
      create/rotate.

### Expiração e lifecycle

- [ ] Boundary `now == expires_at` já é inválida, com teste de relógio
      determinístico.
- [ ] Novos clients não podem omitir expiração nem exceder 365 dias.
- [ ] Revogação vale na request seguinte em qualquer réplica.
- [ ] Após o contract, client revogado não pode ser reativado; a UI orienta
      rotação.

### Rate limit

- [ ] RPM positivo é obrigatório para novos clients; `0/NULL` só existe durante
      compatibilidade legada.
- [ ] Ao exceder, a API retorna `429`, `Retry-After` e headers de limite
      coerentes.
- [ ] Teste com duas instâncias alternando requests prova que o total global não
      excede o bucket compartilhado.
- [ ] Restart de uma réplica não zera o limite global.
- [ ] Indisponibilidade do limiter retorna `503 rate_limiter_unavailable`; não há
      fallback silencioso in-memory.

### Uso, logs e auditoria

- [ ] `last_used_at` avança para autenticação válida, inclusive handler `4xx`, e
      nunca para token inválido/sem scope.
- [ ] Updates concorrentes usam `GREATEST` e não fazem `last_used_at` retroceder.
- [ ] Agregados separam identify/resolve, classe HTTP e recusas `429`.
- [ ] Job remove apenas dias anteriores à retenção de seis meses e expõe falhas.
- [ ] Logs, métricas, usage e auditoria passam por testes-canário que procuram
      token, hash, header e secret values.
- [ ] Create/update/revoke/rotate/delete gera evento admin allowlisted, sem
      segredo.

### API e UI

- [ ] Create/list/edit/revoke/rotate/usage respeitam os contratos acima e
      retornam códigos estáveis.
- [ ] UI explica scopes, exige RPM/expiração, mostra status efetivo e alerta
      `legacy_unbounded`.
- [ ] Rotação permite copiar o sucessor uma vez e apresenta claramente o prazo
      de revogação do anterior.
- [ ] Hard delete não apaga agregados nem eventos históricos.

## Referência comparativa: reference implementation

A inspeção do reference implementation foi somente leitura e serve como evidência comparativa,
não como dependência normativa do Tenancit:

- `apps/apigate/api/internal/store/store.go` modela hash-only, preview, scopes,
  RPM, expiração, revogação, `last_used`, contador e uso diário. A listagem limpa
  o hash; o token é gerado e devolvido uma vez. `Touch` acumula uso e o flush
  persiste deltas, separando o hot path do dashboard.
- `apps/apigate/api/internal/server/auth.go` aplica validade, scope e token bucket
  por ID antes do handler, devolve `403` por scope e `429` por limite. O bucket,
  porém, é um mapa local com mutex: adequado ao reference implementation de instância única,
  inadequado como solução final para Tenancit com múltiplas réplicas.
- `apps/apigate/api/internal/server/admin_clients.go` usa allowlist de scopes,
  expõe token uma vez e separa soft revoke de hard delete.
- `apps/apigate/api/internal/store/login_events.go`,
  `internal/server/admin_metrics.go` e `internal/server/middleware.go` mostram a
  separação útil entre auditoria estruturada, agregados de uso e access log.

### Limites de transferência

Não copiar do reference implementation:

- nomes/prefixos de tokens, scopes de backends ou valores de RPM;
- credenciais, env files, CIDRs, trusted proxies, IPs, topologia de borda, host
  networking ou paths do host;
- SQLite, cache local e tolerâncias de instância única como se fossem garantias
  de um serviço replicado;
- default de scopes amplos quando o operador omite o campo;
- criação sem TTL feita pelo dashboard atual;
- limiter in-memory como controle de produção distribuído.

O que se transfere é o princípio: segredo one-time/hash-only, capabilities
allowlisted, lifecycle visível, limite por identidade e separação entre decisão
de segurança síncrona e telemetria agregada.
