# Auditoria de ações administrativas

- **Status:** Implementado para token compartilhado; OIDC permanece futuro
- **Data:** 2026-07-09
- **Escopo:** trilha append-only das ações na superfície `/v1/admin/*`
- **Relacionados:** ADR 0004 (admin token e API clients), roadmap P1 (login
  humano e auditoria)

## Objetivo

Registrar, de forma consultável e resistente a alteração acidental, quem ou
qual credencial administrativa tentou uma ação, qual ação ocorreu, qual alvo
foi afetado, quando ocorreu, o resultado e o `request_id` que correlaciona o
evento com os logs do processo.

A primeira versão deve cobrir toda mutação administrativa e o reveal de
segredos. Ela não é uma trilha legalmente inviolável, um SIEM ou um substituto
para autenticação humana.

## Princípios

1. **Não inventar identidade.** O `TENANCIT_ADMIN_TOKEN` atual é compartilhado;
   enquanto ele for o login cotidiano da SPA, seu ator é
   `shared_admin_token/admin-token:primary`, não o e-mail de uma pessoa nem um
   evento emergencial fictício.
2. **Somente identidade autenticada pelo servidor.** Um futuro ator humano vem
   de `iss` + `sub` de uma sessão OIDC validada. `X-Actor-Email` e outros headers
   fornecidos pelo cliente são ignorados para atribuição de autoria.
3. **Sem ação sensível não auditada.** Mutações bem-sucedidas gravam o evento na
   mesma transação. Reveal falha antes de devolver cleartext se a auditoria não
   puder ser persistida.
4. **Metadados por allowlist.** Não se serializa request/response inteira. Cada
   ação define explicitamente quais campos não secretos pode registrar.
5. **Append-only com limite declarado.** A aplicação só insere e lê. Trigger e
   privilégios impedem `UPDATE`/`DELETE` de linhas; um administrador do banco
   ainda pode alterar a estrutura, logo isto não é armazenamento WORM.
6. **Histórico sobrevive ao domínio.** Eventos não têm foreign keys para
   tenants/resources/API clients, pois precisam permanecer após hard delete.

## Modelo de ator

O middleware de autenticação deve colocar no contexto um principal verificado,
consumido pelos handlers e pelo gravador de auditoria:

| Campo | Token compartilhado atual | Break-glass futuro | Futuro OIDC |
|---|---|---|---|
| `actor_kind` | `shared_admin_token` | `break_glass` | `oidc_user` |
| `actor_issuer` | `NULL` | `NULL` | valor exato de `iss` validado |
| `actor_subject` | `admin-token:primary` | `admin-token:primary` | valor estável de `sub` |
| `actor_label` | `NULL` | `NULL` | e-mail/nome opcional, apenas snapshot de exibição |
| origem confiável | configuração do servidor | modo emergencial explícito | sessão criada após validar OIDC |

`actor_issuer` + `actor_subject` formam a identidade durável do usuário OIDC.
E-mail não é identificador porque pode mudar ou ser reciclado. O label é dado
pessoal e segue a mesma retenção da auditoria.

Enquanto só existir o token compartilhado, a trilha responde **qual credencial**
agiu, mas não **qual pessoa** a usou. O painel e a documentação não devem afirmar
o contrário. A migração para login humano deve:

- validar assinatura, `iss`, `aud`, expiração e nonce/PKCE no backend;
- criar sessão server-side com cookie `Secure`, `HttpOnly` e `SameSite`;
- exigir autorização por papel/escopo antes do handler;
- preservar `TENANCIT_ADMIN_TOKEN` somente como bootstrap/emergência;
- só mudar seu `actor_kind` para `break_glass` depois de removê-lo do login
  cotidiano e exigir um modo emergencial explícito;
- rotacionar o token por runbook e destacar seu uso emergencial;
- manter a mesma estrutura de eventos, sem reescrever histórico antigo.

Uma evolução com mais de um token emergencial deve dar a cada credencial um ID
de configuração não secreto (`admin-token:primary`, `admin-token:secondary`).
Nunca se registra o token, seu hash ou um prefixo derivado do segredo.

Para uma negativa ocorrida antes da autenticação, o ator é
`unauthenticated/anonymous`, sem issuer ou label. Esse valor não atribui a ação
a uma pessoa e não deriva nenhum identificador da credencial recusada.

## Evento persistido

Tabela proposta: `admin_audit_events`, particionada mensalmente por
`occurred_at`.

| Coluna | Tipo proposto | Regra |
|---|---|---|
| `occurred_at` | `timestamptz` | `NOT NULL DEFAULT clock_timestamp()`; relógio do PostgreSQL |
| `id` | `uuid` | `NOT NULL DEFAULT gen_random_uuid()` |
| `schema_version` | `smallint` | começa em `1`; permite evoluir `metadata` |
| `request_id` | `text` | obrigatório; vindo do middleware, sem unicidade |
| `actor_kind` | `text` | `shared_admin_token`, `break_glass`, `oidc_user`, ou `unauthenticated` em negativas |
| `actor_issuer` | `text` | obrigatório somente para OIDC |
| `actor_subject` | `text` | identificador estável e não secreto |
| `actor_label` | `text` | snapshot opcional para exibição |
| `action` | `text` | nome estável, versionado semanticamente |
| `target_type` | `text` | tipo lógico: `tenant`, `resource`, `api_client`, etc. |
| `target_id` | `text` | UUID ou chave lógica; sem foreign key |
| `result` | `text` | `success`, `denied` ou `error` |
| `http_method` | `text` | método normalizado |
| `route_template` | `text` | template Chi, nunca URL/query string bruta |
| `http_status` | `smallint` | status final esperado pelo handler |
| `error_code` | `text` | código público estável; `NULL` em sucesso |
| `metadata` | `jsonb` | objeto allowlisted, máximo de 8 KiB serializados |

A chave primária é composta por `(occurred_at, id)`, compatível com o
particionamento. `request_id` não é único: uma request pode afetar mais de um
alvo e retries podem produzir novos eventos. A ordenação total usa
`occurred_at DESC, id DESC`.

Checks de banco validam `schema_version > 0`, os três valores de `result`,
campos não vazios e coerência mínima do ator OIDC. Não se criam FKs para tabelas
operacionais. Índices iniciais:

- `(occurred_at DESC, id DESC)` para paginação;
- `(request_id, occurred_at DESC)` para correlação;
- `(target_type, target_id, occurred_at DESC)` para histórico do alvo;
- `(actor_kind, actor_issuer, actor_subject, occurred_at DESC)` para autoria;
- `(action, result, occurred_at DESC)` para investigação.

Índice GIN em `metadata` fica fora da primeira versão: ele aumenta write
amplification e estimula consultas sobre um contrato que deve continuar
pequeno e explícito.

## Vocabulário de ações v1

O nome descreve o efeito de domínio, não o método HTTP. Toda ação registra um
alvo principal e pode citar IDs relacionados somente por allowlist.

### Corte 1 — riscos altos

| Ação | Endpoint atual | Alvo | Metadata permitida |
|---|---|---|---|
| `secret.revealed` | `GET /tenants/{id}/resources?reveal=true` | tenant | `resource_ids`, `secret_field_keys`, contagens; nunca valores |
| `tenant.deleted` | `DELETE /tenants/{id}` | tenant | `slug`, contagem de filhos afetados |
| `resource.deleted` | `DELETE /tenants/{id}/resources/{resourceId}` | resource | `tenant_id`, `definition_id` |
| `definition.field_deleted` | `DELETE /resource-definitions/{id}/fields/{fieldId}` | resource field | `definition_id`, `field_key` |
| `domain.deleted` | `DELETE /tenants/{id}/domains/{domainId}` | domain | `tenant_id`, hostname canônico |
| `api_client.created` | `POST /api-clients` | API client | `name`; nunca token ou `key_hash` |
| `api_client.revoked` | `POST /api-clients/{id}/revoke` | API client | status terminal; nunca token ou hash |
| `resource.provisioned` | `POST /tenants/{id}/resources` | resource | `tenant_id`, `definition_id`, nomes das fields; nunca values |

### Corte 2 — cobertura completa de mutações admin

- `tenant.created` e `tenant.updated`;
- `domain.added`;
- `resource.status_changed`;
- `definition.created`, `definition.status_changed` e
  `definition.field_added`.

A feature só deve ser anunciada como “auditoria admin v1” depois dos dois
cortes. O primeiro corte apenas prioriza revisão e entrega. Leituras comuns não
geram evento, com duas exceções deliberadas: `secret.revealed` e futura
`audit.events_read` para consultas/exportações da própria trilha.

Tentativas bloqueadas pelo middleware podem gerar `admin.auth_denied` com ator
`unauthenticated`, route template e status, sem header, token, IP ou User-Agent.
Essa ação entra depois de rate limit, para não transformar tráfego hostil em
amplificação de escrita no PostgreSQL.

## Redação e dados proibidos

Nunca persistir na auditoria:

- valores de fields secret, mesmo truncados ou mascarados;
- API client token, `key_hash`, `Authorization`, cookies ou sessão;
- `TENANCIT_ADMIN_TOKEN`, chave AES, ciphertext, nonce ou `key_version` quando
  sua presença revelar configuração sensível;
- request/response body completo;
- query string bruta (`?reveal=true` é representado pela ação);
- erro bruto, stack trace, detalhes do PostgreSQL ou SQL;
- IP encaminhado por proxy até existir uma política de trusted proxies.

Cada ação terá um construtor tipado de metadata; handlers não recebem um mapa
JSON irrestrito. O gravador valida tamanho e chaves antes de inserir. Em erro,
registra-se somente um `error_code` público e estável, como
`resource_in_use` ou `internal_error`.

Testes de redação usam canários reconhecíveis nos tokens e secret values e
afirmam que nenhum canário aparece em qualquer coluna ou no JSON serializado.

## Semântica transacional e de resultado

### Mutação bem-sucedida

1. Autenticar e autorizar o principal.
2. Iniciar uma transação PostgreSQL.
3. Executar a mutação com queries vinculadas à transação.
4. Inserir evento `success` usando a mesma transação e IDs já persistidos.
5. Commitar; só então escrever a resposta HTTP.

Se o insert da auditoria falhar, a mutação sofre rollback e o cliente recebe um
erro estável (`503` é o candidato). Isso evita estado administrativo alterado
sem trilha. É uma mudança de boundary: handlers que hoje chamam queries
isoladas precisarão mover a mutação e a auditoria para a mesma transação.

No hard delete, os dados allowlisted necessários ao evento são lidos antes do
delete, mas o evento é inserido depois de a exclusão ter sido validada, ainda na
mesma transação. Cascatas são resumidas por contagens/IDs permitidos, nunca por
snapshots completos.

### Mutação negada ou com erro

Uma transação de domínio que falhou é revertida primeiro. Depois, um insert
curto e separado tenta registrar `denied` ou `error`. Como a operação não
alterou domínio, indisponibilidade da auditoria não muda o resultado original;
o fallback é um log estruturado mínimo com `request_id`, action, target e
`error_code`, sem conteúdo sensível.

O `result` significa:

- `success`: efeito de domínio e evento foram commitados;
- `denied`: autenticação, autorização ou regra de negócio recusou a ação;
- `error`: falha operacional impediu a ação.

Não se registra sucesso antes do commit. Falha ao enviar a resposta depois do
commit não reclassifica o evento: `success` significa que o servidor concluiu o
efeito, não que o browser o exibiu.

### Reveal

Reveal é leitura, mas expõe cleartext. O servidor primeiro valida e monta o
resultado sem escrever o body; em seguida persiste `secret.revealed` em uma
transação curta. Só depois do commit envia os valores. Se a auditoria falhar,
responde erro sem transmitir secrets.

O evento descreve tenant, recursos e chaves dos campos revelados. Ele nunca
contém os values. `success` significa “servidor autorizou, decriptou e ficou
pronto para enviar”; uma desconexão posterior do cliente não pode ser inferida
como leitura humana efetiva.

### Criação de API client

Criação do registro e evento `api_client.created` usam a mesma transação. O
token bruto só é retornado após o commit e nunca entra no evento. Se auditoria
ou commit falhar, nenhum token utilizável é apresentado como criado.

## Garantia append-only no PostgreSQL

A migration de implementação deve:

1. criar tabela pai particionada e partição do mês atual e seguinte;
2. criar checks e índices locais necessários;
3. instalar trigger `BEFORE UPDATE OR DELETE` que sempre lança erro;
4. garantir que o role de runtime tenha somente `INSERT` e `SELECT` na trilha;
5. reservar criação/remoção de partições a um role de migration/manutenção;
6. testar imutabilidade tanto na tabela pai quanto em uma partição.

Hoje o ambiente pode usar uma única credencial de banco. O trigger é o mínimo
obrigatório, mas separação entre role owner/migrator e role de runtime é o
objetivo de produção; sem isso, o próprio owner pode remover a proteção. Mesmo
com roles separados, superusuário PostgreSQL continua fora do threat model.

Não haverá endpoint de update/delete. Correções de evento são novos eventos
(`audit.correction_recorded`) que referenciam o `id` original em metadata.

## API de consulta

Endpoint proposto:

```text
GET /v1/admin/audit-events
  ?from=2026-07-01T00:00:00Z
  &to=2026-07-10T00:00:00Z
  &actor_kind=oidc_user
  &actor_subject=...
  &action=tenant.deleted
  &target_type=tenant
  &target_id=...
  &request_id=...
  &result=success
  &cursor=...
  &limit=50
```

Regras:

- admin token pode consultar durante a transição; após OIDC, exige permissão
  `audit.read`, separada da permissão de mutar tenants;
- padrão de 24 horas, janela máxima de 31 dias por request e `limit` máximo 200;
- paginação keyset por `(occurred_at, id)`, sem offset;
- filtros são exatos; busca livre em metadata fica fora da v1;
- resposta nunca enriquece eventos consultando tabelas atuais, pois o alvo pode
  ter sido removido;
- exportação em massa e streaming para SIEM ficam fora do escopo;
- a própria consulta gera `audit.events_read` sem incluir resultados, apenas
  filtros allowlisted e quantidade retornada.

## Retenção, partições e volume

Proposta inicial: partições mensais e retenção online de 180 dias, a confirmar
com requisitos legais e operacionais antes da implementação. Uma rotina
privilegiada cria partições antecipadamente e remove a partição inteira já
expirada; a aplicação nunca executa `DELETE` linha a linha.

Antes de remover uma partição, a rotina registra em uma partição vigente o
período removido, contagem de linhas e identificador do backup. Backup e restore
da trilha entram no runbook operacional. Legal hold e retenção imutável externa
não fazem parte desta proposta.

Dimensionamento deve usar telemetria real. Para planejamento, considere cerca
de 1–2 KiB por evento com índices, sem contar WAL e backups:

- 1.000 ações/dia: aproximadamente 30–60 MiB/mês;
- 10.000 ações/dia: aproximadamente 300–600 MiB/mês.

Alertas devem cobrir falha de insert, ausência de partição futura, crescimento
fora do esperado e atraso da rotina de retenção.

## Estratégia de migração e entrega

1. **Migration aditiva:** criar tabela, partições, trigger e índices. Não há
   backfill; histórico anterior ao deploy é desconhecido e não deve ser
   sintetizado.
2. **Primitivas internas:** principal autenticado no contexto, tipo fechado de
   ação/metadata e gravador que aceite `pgx.Tx`.
3. **Corte 1:** instrumentar reveal, hard deletes, API clients e provisionamento;
   habilitar métricas de insert/erro/latência.
4. **Corte 2:** instrumentar todas as demais mutações e adicionar a consulta
   paginada.
5. **Operação:** criar job de partições/retenção, backup/restore e alertas.
6. **OIDC/RBAC:** trocar a origem do principal sem alterar eventos antigos;
   manter break-glass com uso destacado e alertável.

O deploy pode aplicar a migration antes do binário novo: versões antigas apenas
deixam a tabela vazia. Depois de ativada a auditoria fail-closed, rollback para
binário sem instrumentação deve exigir decisão operacional explícita, porque
voltaria a permitir ações sem eventos.

## Critérios de aceite e evidência de implementação

- [ ] Migration sobe e desce em banco descartável; update/delete de evento falha.
- [ ] Role de runtime não consegue alterar ou remover linhas de auditoria.
- [ ] Cada endpoint mutável de `/v1/admin/*` produz exatamente um evento de
      sucesso por alvo principal.
- [ ] Falha forçada no insert de auditoria reverte a mutação de domínio.
- [ ] Falha forçada na auditoria de reveal devolve erro e zero bytes de secrets.
- [ ] API client recém-criado nunca deixa token ou hash na trilha.
- [ ] Canários de secret, token, header e erro bruto não aparecem no banco.
- [ ] `X-Actor-Email` não altera autoria; OIDC usa somente claims validadas.
- [ ] O token cotidiano aparece como `shared_admin_token`; somente o modo
      emergencial explícito aparece como `break_glass`, ambos sem atribuição humana.
- [ ] Filtros e paginação keyset são estáveis sob inserts concorrentes.
- [ ] Eventos permanecem consultáveis após hard delete do alvo.
- [ ] Partição futura, retenção, backup e restore têm testes/runbook.
- [ ] Métricas distinguem sucesso, falha e latência de escrita sem labels de
      alta cardinalidade ou dados sensíveis.

## Não objetivos

- reconstrução completa de estado ou event sourcing;
- storage WORM, assinatura criptográfica, legal hold ou prova forense contra o
  administrador do PostgreSQL;
- auditoria de toda leitura comum;
- UI analítica completa;
- exportação SIEM em tempo real;
- captura de bodies, valores secretos ou diffs genéricos;
- implementação de login humano neste spike.

## Questões abertas para aprovação

1. A retenção online de 180 dias atende requisitos legais e de suporte? Há
   necessidade de arquivo externo por prazo maior?
2. Qual IdP e quais claims/grupos autorizam `audit.read`, reveal e hard delete?
3. O break-glass continuará com uma credencial ou precisará de IDs distintos
   para rotação sem perda de rastreabilidade?
4. Consultas à trilha devem registrar apenas contagem/filtros ou também um
   motivo operacional informado pelo usuário?
5. IP e User-Agent têm valor investigativo suficiente para justificar coleta,
   política de trusted proxies e tratamento de dados pessoais?
6. Qual orçamento de latência e disponibilidade deve acionar circuit breaker
   ou modo somente leitura do admin?
7. Existe requisito futuro de exportação para SIEM/WORM que mude a escolha de
   partições e retenção antes da primeira migration?

## Decisão recomendada

Aprovar o modelo e implementar em cortes: primeiro as ações de maior risco,
depois cobertura integral e consulta. Tratar `TENANCIT_ADMIN_TOKEN` como
`shared_admin_token` enquanto ele for cotidiano; migrá-lo para `break_glass`
somente quando OIDC/RBAC retirarem o token da SPA e o modo emergencial for
explícito. Preservar fail-closed em mutações bem-sucedidas e reveal, porque uma
auditoria opcional não satisfaz o objetivo do roadmap.
