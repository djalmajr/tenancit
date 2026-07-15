# Usability — Investigar, preservar e exportar atividade administrativa (audit-operations)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `partial`

## Evidência

As abas Visão geral/Eventos, cards, busca, filtro de resultado, ordenação, colunas e paginação foram exercitados; buscar `admin.request_failed` reduziu o conjunto a um evento.

## Limite ou achado

Exportação e legal hold não foram acionados por serem mutações/artefatos operacionais.

## Próximo passo

Rodar export/hold em fixture isolada e conferir auditoria da própria ação.
