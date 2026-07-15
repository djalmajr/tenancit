# Usability — Reutilizar configuração por vínculo, override e duplicação (tenant-shared-resource-inheritance)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `blocked`

## Evidência

O diálogo Adicionar recurso apresentou alias único e Origem dos valores com opção Independente, confirmando a entrada do modelo atual.

## Limite ou achado

Não existia recurso vinculado preparado para provar propagação, override, duplicação independente e bloqueio de remoção da origem.

## Próximo passo

Criar fixture origem+vínculo com um override e repetir toda a matriz de herança.
