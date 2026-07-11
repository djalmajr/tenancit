# Backup e restore PostgreSQL

**Status:** VALIDADO em 2026-07-09 com dump custom, restore em
`tenancit_restore_check`, verificação de dado sentinela e remoção do banco isolado.

O Compose usa o volume nomeado `tenancit-postgres-data`. `docker compose down`
preserva esse volume; `down -v` e `make docker-reset` o removem.

Execute todos os snippets a partir da raiz do checkout; eles usam `pwd -P` para
impedir que dumps sejam gravados dentro do repositório.

## Backup

```bash
set -euo pipefail
: "${TENANCIT_BACKUP_DIR:?set an absolute backup directory outside this checkout}"
case "$TENANCIT_BACKUP_DIR" in
  /*) ;;
  *) echo "TENANCIT_BACKUP_DIR must be absolute" >&2; exit 1 ;;
esac
umask 077
mkdir -p "$TENANCIT_BACKUP_DIR"
repo_root="$(pwd -P)"
backup_dir="$(cd "$TENANCIT_BACKUP_DIR" && pwd -P)"
case "$backup_dir/" in
  "$repo_root/"*) echo "backup directory must be outside the checkout" >&2; exit 1 ;;
esac
dump="$backup_dir/tenancit-$(date -u +%Y%m%dT%H%M%SZ)-$$.dump"
trap 'rm -f "$dump"' ERR INT TERM
docker compose exec -T postgres \
  pg_dump -U postgres -d tenancit -Fc > "$dump"
docker compose exec -T postgres pg_restore --list < "$dump" >/dev/null
trap - ERR INT TERM
echo "validated dump: $dump"
```

`TENANCIT_BACKUP_DIR` deve apontar para storage protegido fora do checkout. O
comando valida exatamente o arquivo recém-criado e usa o `pg_restore` do próprio
container, sem exigir cliente PostgreSQL no host. Guarde o dump criptografado
fora do host. Ele contém ciphertext e metadados; as chaves
`TENANCIT_AES_KEY*` ficam no secret manager e precisam de backup separado.

## Ensaio de restore isolado

Use um banco novo, sem sobrescrever o atual:

```bash
set -euo pipefail
: "${TENANCIT_BACKUP_FILE:?set the absolute dump path to restore}"
case "$TENANCIT_BACKUP_FILE" in
  /*) ;;
  *) echo "TENANCIT_BACKUP_FILE must be absolute" >&2; exit 1 ;;
esac
test -f "$TENANCIT_BACKUP_FILE"
repo_root="$(pwd -P)"
dump_dir="$(cd "$(dirname "$TENANCIT_BACKUP_FILE")" && pwd -P)"
case "$dump_dir/" in
  "$repo_root/"*) echo "backup file must be outside the checkout" >&2; exit 1 ;;
esac
docker compose exec -T postgres pg_restore --list < "$TENANCIT_BACKUP_FILE" >/dev/null
restore_db="tenancit_restore_check_$(date -u +%Y%m%d%H%M%S)_$$"
cleanup_restore() {
  docker compose exec -T postgres dropdb -U postgres --if-exists "$restore_db" >/dev/null 2>&1 || true
}
trap cleanup_restore EXIT INT TERM
docker compose exec -T postgres createdb -U postgres "$restore_db"
docker compose exec -T postgres pg_restore \
  -U postgres -d "$restore_db" --no-owner --no-privileges < "$TENANCIT_BACKUP_FILE"
docker compose exec -T postgres psql -U postgres -d "$restore_db" \
  -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
cleanup_restore
trap - EXIT INT TERM
```

## Restore de produção

1. interrompa writers e capture um backup final;
2. restaure em um banco novo;
3. execute o binário Tenancit contra o novo DSN para aplicar migrations;
4. rode o [smoke pós-deploy](post-deploy-smoke.md);
5. só então troque o tráfego.

Rollback: volte o DSN para o banco anterior, mantido intacto até o término da
janela de observação. Nunca restaure diretamente por cima da única cópia.
