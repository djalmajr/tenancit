# Usability — Criar webhook e acompanhar entregas (webhook-target-and-delivery-operations)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `blocked`

## Evidência

Webhooks abriu com abas Destinos/Entregas, tabelas padronizadas e diálogo Novo webhook com nome, URL e formato.

## Limite ou achado

Não havia receptor de teste, destino nem entrega; criar alvo externo fugiria da fixture segura desta rodada.

## Próximo passo

Subir receiver local E2E e validar segredo one-shot, retry e dead letter.
