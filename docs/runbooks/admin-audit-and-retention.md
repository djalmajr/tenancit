# Auditoria administrativa e retenção

**Status:** VALIDADO em PostgreSQL descartável.

Eventos de sucesso de mutações e reveal participam da mesma transação do domínio.
Falha no insert reverte a operação e nenhum cleartext é enviado. Negativas e
erros são registrados best-effort depois da resposta para não alterar o erro
original. Metadata é allowlisted e não recebe body, query string ou headers.

## Operação

- Consulte `/v1/admin/audit-events` com janela padrão de 24 h, máximo de 31 dias,
  limite até 200 e cursor keyset.
- O ator `shared_admin_token/primary` identifica a credencial, não uma pessoa.
- A trigger bloqueia `UPDATE` e `DELETE`; conceda ao runtime somente `INSERT` e
  `SELECT`, mantendo DDL/partições no role de migration.
- Crie partições mensais futuras antes da virada do mês e monitore falhas de
  insert. Exporte partições antes da retenção definida pela organização.
- Backup/restore deve preservar eventos mesmo depois do hard delete do alvo.

Use canários nos testes para confirmar ausência de tokens, hashes, cookies,
secrets, bodies e query strings. Um evento de correção deve ser novo e
append-only; nunca altere o evento original.
