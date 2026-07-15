# Usability — Repetir mutações administrativas críticas com segurança (admin-idempotent-retry)

- **Data:** 2026-07-15
- **Ambiente:** Browser integrado · `http://localhost:5180/`
- **Veredito:** `blocked`

## Evidência

O fluxo é predominantemente HTTP e não possui controle visual para repetir a mesma mutation key.

## Limite ou achado

O Browser integrado, por navegação de UI, não prova respostas replayed nem conflito de payload.

## Próximo passo

Manter a prova nos testes de integração HTTP e expor correlação apenas se houver valor operacional.
