---
id: api-client-usage-audit
name: Investigar uso e auditoria de chaves
persona: platform-operator
entry: "http://localhost:5180/usage"
preconditions:
  - app no ar e operador autenticado
  - ao menos uma chamada consumer realizada
---

## Objetivo do usuário

Relacionar atividade operacional de uma chave às ações administrativas que definiram seu lifecycle.

## Passos

1. (`usage`) Abrir **Uso mensal** → cards e série mensal mostram chamadas, erros e limitações sem expor secrets.
2. (`usage`) Filtrar por chave, operação e período → apenas os agregados correspondentes permanecem.
3. (`audit-events`) Abrir **Auditoria** → eventos exibem ator-credencial, ação, alvo, resultado e request ID.
4. (`audit-events`) Filtrar pela ação e pelo alvo da chave → criação, edição, rotação e revogação são localizáveis.

## Resultado esperado

O operador consegue investigar uso e mutações sem tokens, hashes, bodies, cookies ou query strings.
