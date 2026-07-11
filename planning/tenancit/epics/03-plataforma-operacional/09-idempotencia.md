# História 09 — Idempotência administrativa

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Timeouts e retries não podem duplicar tenant, client sucessor, resource ou
webhook. Implementar contrato uniforme para mutações selecionadas.

## Responsabilidade, motivação e valor

Uma resposta pode se perder depois do commit e ser repetida pelo navegador,
proxy ou automação. Em operações sensíveis isso poderia criar dois tokens
sucessores, dois recursos ou efeitos divergentes sem o operador saber qual vale.

**Ganho:** retries seguros e exatamente um efeito observável nas mutações
críticas. A aplicação é seletiva — create, rotate e provision primeiro — e não
transforma toda leitura ou edição simples em workflow idempotente.

## Arquivos

- Middleware/store de idempotency records, migration e cleanup.
- Aplicação nos endpoints create/rotate/provision e testes/documentação.

## Detalhe

Chave é escopada por principal + operação, request fingerprint e TTL. Mesmo
fingerprint reproduz status/resultado sanitizado; payload diferente retorna
conflito. Processamento concorrente tem um único owner e wait/409 previsível.
Secrets one-shot exigem desenho específico: retry autorizado pode recuperar o
mesmo envelope cifrado por janela curta, nunca criar novo sucessor.

## Tarefas

- [x] Definir endpoints, TTL, fingerprint e respostas estáveis.
- [x] Criar schema/cleanup e execução transacional com domínio/outbox/audit.
- [x] Implementar create, rotate e provision prioritários.
- [x] Cobrir concorrência, crash pós-commit e mismatch de payload.
- [x] Atualizar SDK snippets, docs e E2E de retry.

## Verificação

Testes concorrentes e fault injection provam um efeito; nenhum secret bruto é
persistido fora do envelope/janela explicitamente aprovada.

## Evidência entregue

- `admin_idempotency_records` escopa UUID por `actor + operation`, vincula
  fingerprint SHA-256 e guarda somente a resposta AES-GCM; runtime não pode
  apagar claims e o worker privilegiado remove expirados em lotes.
- Create tenant e provision possuem TTL de 24 h; create/rotate de API client
  usam 10 min e devolvem exatamente o mesmo token no replay. Payload diferente
  recebe `409 idempotency_mismatch`.
- Claim, domínio, outbox, auditoria e envelope são confirmados juntos. Rollback
  libera a chave; concorrência espera o owner e observa um único resultado.
- A SPA preserva a chave enquanto o payload não muda e gera outra após edição.
  O catálogo `admin-idempotent-retry` valida tenant, token, rotação,
  provisionamento e mismatch no produto empacotado.
