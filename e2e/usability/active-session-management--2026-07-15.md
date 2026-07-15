# Usability — Investigar e revogar sessões ativas (active-session-management)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `blocked`

## Evidência

A tela de Sessões ativas abriu com busca, ordenação, colunas e paginação; o conjunto estava vazio.

## Limite ou achado

Sem uma segunda sessão OIDC não foi possível validar proteção da sessão corrente nem revogação imediata.

## Próximo passo

Preparar duas sessões humanas em modo OIDC e repetir os passos 2–6.
