---
id: active-session-management
name: Investigar e revogar sessões ativas
reference: web/src/routes/sessions.page.tsx
persona: administrator
entry: "http://localhost:5180/"
status: draft
preconditions:
  - app no ar em modo OIDC e pessoa administradora autenticada com `session.manage`
  - existe ao menos uma segunda sessão ativa revogável
---

## Objetivo do usuário

Descobrir quem está autenticado no console e encerrar imediatamente uma sessão suspeita sem derrubar a sessão atual por engano.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`sessions`) Na entrada, clicar em **Sessões ativas** na sidebar → a tabela abre com principal, papéis, último uso, expiração, status e ação.
2. (`sessions`) Buscar um principal e ordenar por último uso → a tabela reduz e ordena o conjunto sem perder contexto de sessão atual.
3. (`sessions`) Abrir **Colunas**, alterar visibilidade e restaurar preferências → a escolha persiste no browser e pode ser revertida.
4. (`sessions`) Conferir a sessão marcada como atual → ela não oferece a ação de auto-revogação na tabela.
5. (`revoke`) Clicar no ícone de encerrar de outra sessão → a confirmação identifica issuer e subject afetados.
6. (`revoke`) Confirmar **Revogar** → a sessão deixa de ser válida imediatamente e a tabela reflete o novo status ou sua remoção.

## Resultado esperado

A pessoa administradora encontra e revoga somente a sessão pretendida, com confirmação contextual e proteção contra encerramento acidental da sessão corrente.
