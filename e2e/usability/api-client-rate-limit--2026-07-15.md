# Usability — Diagnosticar limite global de uma chave (api-client-rate-limit)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `blocked`

## Evidência

A tela de Uso mensal mostrou métricas de limitadas e filtros por chave/operação.

## Limite ou achado

Não havia token bruto nem chave descartável de RPM baixo para gerar `429` pelo Browser.

## Próximo passo

Criar fixture de chave limitada e validar headers/códigos no teste HTTP multi-instância.
