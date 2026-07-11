# Auditoria administrativa, exportação e retenção

**Status:** VALIDADO em PostgreSQL descartável em 2026-07-11.

## Contratos

- Consultas: `GET /v1/admin/audit-events`, janela padrão 24 h, máxima 31 dias,
  limite máximo 200 e cursor keyset. Exige `audit.read` e gera
  `audit.events_read` com janela/contagem, nunca query string ou resultados.
- Export: `POST /v1/admin/audit-exports` exige `audit.export`, UUID em
  `Idempotency-Key`, filtros exatos, CSV/JSONL e no máximo 100.000 eventos.
  Até 1.000 é materializado na própria request; acima disso o job dedicado
  processa assincronamente.
- Download: `GET /v1/admin/audit-exports/{id}/download` é one-shot, `no-store`,
  expira em 24 h e grava `audit.export_downloaded` na mesma transação que apaga
  o ciphertext. O endpoint de status nunca retorna payload, nonce ou key version.
- Legal hold: `/v1/admin/audit-legal-holds` exige `audit.manage`; criação e
  liberação são auditadas e uma janela ativa impede o descarte da partição.

## Job e partições

Execute `/tenancit-audit-jobs` com `TENANCIT_JOBS_DATABASE_URL` do login mínimo
`tenancit_jobs` e o mesmo keyring AES da aplicação. O processo:

1. drena a partição default sob advisory + `ACCESS EXCLUSIVE` lock, roteando
   cada mês sem alterar eventos;
2. mantém o mês atual e três meses futuros materializados;
3. descarta partições além de `audit_retention_days`, exceto as que intersectam
   legal hold ativo;
4. processa jobs com `FOR UPDATE SKIP LOCKED` e apaga artefatos expirados.

A função DDL é `SECURITY DEFINER`, tem nome/intervalos derivados internamente e
somente `tenancit_jobs` recebe `EXECUTE`. O runtime apenas consulta saúde e
opera holds/jobs em colunas explicitamente concedidas. Alerta se
`current_month_covered=false`, se `default_rows>0` persistir após um ciclo, ou se
o worker `audit_retention|audit_export` registrar erro.

## Incidente e WORM/SIEM

Antes de investigar, crie legal hold com janela e referência do incidente. Para
arquivo externo, implemente `auditops.ArchiveSink`; o adapter recebe metadata,
SHA-256 e bytes somente durante a geração. Falha do destino deixa o job `failed`
e nenhum arquivo local pronto. Não há vendor padrão nem fallback silencioso.

Use canários para confirmar ausência de tokens, hashes, cookies, secrets,
bodies e query strings. Backup/restore deve preservar eventos após hard delete.
Correções são novos eventos; nunca altere o original.
