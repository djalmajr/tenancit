---
id: webhook-target-and-delivery-operations
name: Criar webhook e acompanhar entregas
reference: web/src/routes/integrations.page.tsx
persona: administrator
entry: "http://localhost:5180/"
status: draft
preconditions:
  - app no ar e pessoa administradora autenticada com `integration.manage`
  - existe um receiver HTTPS controlado para validar assinatura e entregas
---

## Objetivo do usuário

Cadastrar um destino assinado, guardar o segredo one-shot e acompanhar entregas, retentativas e dead letters pela mesma área operacional.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`targets`) Na entrada, clicar em **Webhooks** na sidebar → a aba segmentada **Destinos** abre com busca, ordenação, colunas e paginação consistentes.
2. (`create`) Clicar em **Novo webhook**, preencher nome, URL HTTPS e formato e salvar → o destino é criado sem expor detalhes internos da fila.
3. (`secret`) Copiar o segredo de assinatura exibido no diálogo one-shot e concluir → o segredo não volta a aparecer na tabela.
4. (`targets`) Buscar o destino e alternar seu status → a ação e o badge deixam claro se novas entregas estão habilitadas.
5. (`deliveries`) Selecionar **Entregas** → a tabela mostra evento, destino, status, tentativas e ações sem misturar configuração de destinos.
6. (`deliveries`) Buscar uma entrega e inspecionar seu estado → sucesso, pendência, retry e dead letter são distinguíveis.
7. (`dead-letter`) Acionar a repetição de uma entrega em dead letter → a UI confirma a solicitação e a entrega volta ao ciclo operacional sem duplicar o destino.

## Resultado esperado

A pessoa administradora configura o webhook, preserva o segredo de assinatura e investiga o lifecycle de entregas com contexto suficiente para recuperação.
