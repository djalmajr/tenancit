# Usability — Configurar tenant no admin e consumir pelo caminho seguro identify → alias (admin-to-consumer-golden-path)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `partial`

## Evidência

O tenant MyDesk, seus domínios, recursos e a chave ativa foram inspecionados pelas telas do console.

## Limite ou achado

A metade consumer (`identify` e resolução por alias) é uma jornada HTTP sem superfície visual; nenhuma credencial descartável foi criada.

## Próximo passo

Combinar a rodada Browser com o teste de contrato consumer usando fixture própria.
