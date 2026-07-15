---
id: api-client-governance
name: Governar política e lifecycle de uma chave de API
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar e operador autenticado
---

## Objetivo do usuário

Criar uma credencial com menor privilégio, editar sua política, rotacionar o segredo e encerrar seu lifecycle com revogação terminal.

## Passos

1. (`api-clients`) Na entrada, clicar em **Chaves de API** na sidebar e criar uma chave com scope explícito, RPM positivo e expiração futura → o token é exibido uma única vez.
2. (`api-clients`) Editar nome, scopes, RPM e expiração → a tabela reflete a política atualizada.
3. (`api-clients`) Rotacionar a chave → um sucessor one-shot é exibido e a janela de transição fica explícita.
4. (`api-clients`) Revogar a chave → o token deixa de autenticar imediatamente e não existe ação de reativação.
5. (`api-clients`) Remover a chave revogada → ela sai do inventário, mas uso e auditoria permanecem.

## Resultado esperado

Nenhuma credencial nasce ilimitada, material secreto nunca reaparece na listagem e revogação é terminal.
