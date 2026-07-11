# História 09 — Idempotência administrativa

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Timeouts e retries não podem duplicar tenant, client sucessor, resource ou
webhook. Implementar contrato uniforme para mutações selecionadas.

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

- [ ] Definir endpoints, TTL, fingerprint e respostas estáveis.
- [ ] Criar schema/cleanup e execução transacional com domínio/outbox/audit.
- [ ] Implementar create, rotate e provision prioritários.
- [ ] Cobrir concorrência, crash pós-commit e mismatch de payload.
- [ ] Atualizar SDK snippets, docs e E2E de retry.

## Verificação

Testes concorrentes e fault injection provam um efeito; nenhum secret bruto é
persistido fora do envelope/janela explicitamente aprovada.

