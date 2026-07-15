---
id: audit-operations
name: Investigar, preservar e exportar atividade administrativa
reference: web/src/routes/audit-events.page.tsx
persona: auditor
entry: "http://localhost:5180/"
preconditions:
  - app no ar com token administrativo válido
---

## Objetivo do usuário

Investigar atividade com filtros confiáveis, preservar uma janela sob incidente e obter um artefato controlado sem expor material permanente.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`activity`) Na entrada, clicar em **Auditoria** e alternar entre **Visão geral** e **Eventos** → saúde das partições e trilha ficam separadas sem exigir acesso ao banco.
2. (`filters`) Informar uma ação e aplicar filtros → a request usa filtro server-side e a tabela mostra somente o recorte.
3. (`legal-hold`) Abrir **Gerenciar legal holds**, criar uma janela com referência e liberá-la → o estado ativo/liberado fica explícito e auditável.
4. (`export`) Solicitar **Exportar trilha** e baixar quando pronto → o browser recebe CSV uma vez e a ação de download desaparece após o consumo.

## Resultado esperado

O auditor percorre consulta, preservação e exportação pela UI, com paginação/filtros no servidor, expiração e download one-shot.
