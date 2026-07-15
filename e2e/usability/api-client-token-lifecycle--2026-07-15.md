# Usability — Gerar, copiar, revogar e remover chave de API (api-client-token-lifecycle)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `partial`

## Evidência

A tela e o diálogo de criação foram inspecionados; o token bruto não aparece na listagem.

## Limite ou achado

Create/rotate/revoke/delete não foram acionados para preservar a chave operacional única.

## Próximo passo

Executar sobre chave descartável e comprovar o one-shot do segredo.
