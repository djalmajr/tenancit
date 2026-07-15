---
id: api-client-rate-limit
name: Diagnosticar limite global de uma chave
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app e Valkey disponíveis
  - chave com RPM baixo criada para o cenário
---

## Objetivo do usuário

Reconhecer uma recusa por limite, aguardar a janela indicada e confirmar a visibilidade operacional do evento.

## Passos

1. (`consumer`) Na entrada, clicar em **Chaves de API**, selecionar a credencial de RPM baixo e consumir uma rota autorizada com ela até esgotar o bucket → a próxima chamada retorna `429 rate_limited`.
2. (`consumer`) Inspecionar a resposta → `Retry-After` e headers `RateLimit-*` informam limite e recuperação.
3. (`usage`) Pela sidebar, abrir **Uso mensal** → a limitação aparece separada dos demais erros.
4. (`health`) Pela sidebar, abrir **Saúde** e confirmar a dependência externa saudável; o teste de integração de indisponibilidade prova `503 rate_limiter_unavailable`, sem fallback local.

## Resultado esperado

O limite é global por client ID, observável e falha fechado sem armazenar token ou hash no Valkey.
