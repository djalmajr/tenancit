---
id: admin-idempotent-retry
name: Repetir mutações administrativas críticas com segurança
reference: planning/tenancit/epics/03-plataforma-operacional/09-idempotencia.md
persona: administrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar com PostgreSQL e token administrativo válidos
---

## Objetivo do usuário

Repetir uma mutação após timeout sem criar um segundo efeito ou perder o token one-shot já confirmado.

## Passos (cada passo é uma AÇÃO + o resultado esperado)

1. (`admin-api`) Reenviar a criação de tenant com a mesma `Idempotency-Key` e payload → a resposta retorna o mesmo tenant e marca replay, sem duplicar linhas ou eventos.
2. (`admin-api`) Reutilizar a chave com payload diferente → a API retorna `409 idempotency_mismatch` e não altera o domínio.
3. (`admin-api`) Reenviar create e rotate de API client → cada retry recupera exatamente o mesmo token cifrado dentro da janela curta, sem criar sucessor adicional.
4. (`admin-api`) Reenviar provisionamento de resource → a resposta retorna a mesma instância e permanece somente um resource ativo.

## Resultado esperado

O operador pode repetir operações críticas com um único efeito observável; a chave não mascara divergência de payload e secrets não ficam persistidos em claro.
