#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
namespace="${TENANCIT_K8S_NAMESPACE:-tenancit}"
release="${TENANCIT_K8S_RELEASE:-tenancit}"
values="$root_dir/deploy/helm/tenancit/values-personal.yaml"
secret_name="tenancit-runtime"
state_dir="${TENANCIT_PERSONAL_STATE_DIR:-$HOME/.config/tenancit}"
state_file="$state_dir/personal.env"
: "${TENANCIT_IMAGE_DIGEST:?set the published immutable sha256 digest}"
case "$TENANCIT_IMAGE_DIGEST" in sha256:[0-9a-f][0-9a-f]*) ;; *) echo "invalid TENANCIT_IMAGE_DIGEST" >&2; exit 1;; esac
[ "${#TENANCIT_IMAGE_DIGEST}" -eq 71 ] || { echo "invalid TENANCIT_IMAGE_DIGEST length" >&2; exit 1; }

command -v kubectl >/dev/null
command -v helm >/dev/null
command -v openssl >/dev/null
command -v htpasswd >/dev/null
command -v jq >/dev/null
mkdir -p "$state_dir"
chmod 700 "$state_dir"

random_hex() { openssl rand -hex "$1"; }
if [ ! -f "$state_file" ]; then
  admin_password="$(random_hex 18)"
  cat >"$state_file" <<EOF
POSTGRES_ADMIN_PASSWORD=$(random_hex 24)
POSTGRES_MIGRATION_PASSWORD=$(random_hex 24)
POSTGRES_RUNTIME_PASSWORD=$(random_hex 24)
POSTGRES_JOBS_PASSWORD=$(random_hex 24)
POSTGRES_BACKUP_PASSWORD=$(random_hex 24)
POSTGRES_REWRAP_PASSWORD=$(random_hex 24)
VALKEY_PASSWORD=$(random_hex 24)
OIDC_CLIENT_SECRET=$(random_hex 32)
AES_KEY=$(openssl rand -base64 32 | tr -d '\n')
OPERATIONS_TOKEN=$(random_hex 32)
ADMIN_EMAIL=admin@tenancit.local
ADMIN_USERNAME=tenancit-admin
ADMIN_PASSWORD=$admin_password
EOF
  chmod 600 "$state_file"
fi
. "$state_file"

digest_hex="${TENANCIT_IMAGE_DIGEST#sha256:}"
case "$digest_hex" in *[!0-9a-f]*) echo "invalid TENANCIT_IMAGE_DIGEST characters" >&2; exit 1;; esac

host="${TENANCIT_PUBLIC_HOST:-tenancit.djalmajr.dev}"
oidc_host="${TENANCIT_OIDC_HOST:-tenancit-id.djalmajr.dev}"
pg_service="$release-postgres.$namespace.svc.cluster.local"
valkey_service="$release-valkey.$namespace.svc.cluster.local"
urlencode() { jq -nr --arg v "$1" '$v|@uri'; }
admin_q="$(urlencode "$POSTGRES_ADMIN_PASSWORD")"
migration_q="$(urlencode "$POSTGRES_MIGRATION_PASSWORD")"
runtime_q="$(urlencode "$POSTGRES_RUNTIME_PASSWORD")"
jobs_q="$(urlencode "$POSTGRES_JOBS_PASSWORD")"
valkey_q="$(urlencode "$VALKEY_PASSWORD")"
dex_hash="$(htpasswd -bnBC 10 '' "$ADMIN_PASSWORD" | tr -d ':\n')"

kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
dex_config="$(cat <<EOF
issuer: https://$oidc_host/dex
storage:
  type: sqlite3
  config:
    file: /var/dex/dex.db
web:
  http: 0.0.0.0:5556
oauth2:
  skipApprovalScreen: true
staticClients:
  - id: tenancit
    name: Tenancit
    secret: $OIDC_CLIENT_SECRET
    redirectURIs:
      - https://$host/v1/auth/callback
enablePasswordDB: true
staticPasswords:
  - email: $ADMIN_EMAIL
    hash: '$dex_hash'
    username: $ADMIN_USERNAME
    userID: 66b9f7dc-2fe5-45f4-aea2-d52b687c12a1
EOF
)"

secret_value() {
  kubectl -n "$namespace" get secret "$secret_name" -o json |
    jq -r --arg key "$1" '.data[$key] // empty' | base64 -d
}
check_secret_value() {
  [ "$(secret_value "$1")" = "$2" ] || {
    echo "credential drift for $1; use an explicit credential-rotation procedure" >&2
    exit 1
  }
}
if kubectl -n "$namespace" get secret "$secret_name" >/dev/null 2>&1; then
  check_secret_value postgres-admin-password "$POSTGRES_ADMIN_PASSWORD"
  check_secret_value postgres-migration-password "$POSTGRES_MIGRATION_PASSWORD"
  check_secret_value postgres-runtime-password "$POSTGRES_RUNTIME_PASSWORD"
  check_secret_value postgres-jobs-password "$POSTGRES_JOBS_PASSWORD"
  check_secret_value postgres-backup-password "$POSTGRES_BACKUP_PASSWORD"
  check_secret_value postgres-rewrap-password "$POSTGRES_REWRAP_PASSWORD"
  check_secret_value valkey-password "$VALKEY_PASSWORD"
  check_secret_value oidc-client-secret "$OIDC_CLIENT_SECRET"
  check_secret_value aes-key "$AES_KEY"
  check_secret_value operations-token "$OPERATIONS_TOKEN"
fi

b64() { printf '%s' "$1" | base64 | tr -d '\n'; }
printf '%s\n' "{\"apiVersion\":\"v1\",\"kind\":\"Secret\",\"metadata\":{\"name\":\"$secret_name\",\"namespace\":\"$namespace\"},\"type\":\"Opaque\",\"data\":{
\"postgres-admin-password\":\"$(b64 "$POSTGRES_ADMIN_PASSWORD")\",
\"postgres-migration-password\":\"$(b64 "$POSTGRES_MIGRATION_PASSWORD")\",
\"postgres-runtime-password\":\"$(b64 "$POSTGRES_RUNTIME_PASSWORD")\",
\"postgres-jobs-password\":\"$(b64 "$POSTGRES_JOBS_PASSWORD")\",
\"postgres-backup-password\":\"$(b64 "$POSTGRES_BACKUP_PASSWORD")\",
\"postgres-rewrap-password\":\"$(b64 "$POSTGRES_REWRAP_PASSWORD")\",
\"migration-database-url\":\"$(b64 "postgres://tenancit_migration:$migration_q@$pg_service:5432/tenancit?sslmode=disable")\",
\"runtime-database-url\":\"$(b64 "postgres://tenancit_runtime_login:$runtime_q@$pg_service:5432/tenancit?sslmode=disable")\",
\"jobs-database-url\":\"$(b64 "postgres://tenancit_jobs_login:$jobs_q@$pg_service:5432/tenancit?sslmode=disable")\",
\"backup-database-url\":\"$(b64 "postgres://tenancit_backup_login:$(urlencode "$POSTGRES_BACKUP_PASSWORD")@$pg_service:5432/tenancit?sslmode=disable")\",
\"rewrap-database-url\":\"$(b64 "postgres://tenancit_rewrap_login:$(urlencode "$POSTGRES_REWRAP_PASSWORD")@$pg_service:5432/tenancit?sslmode=disable")\",
\"postgres-admin-url\":\"$(b64 "postgres://postgres:$admin_q@$pg_service:5432/tenancit?sslmode=disable")\",
\"valkey-password\":\"$(b64 "$VALKEY_PASSWORD")\",
\"valkey-url\":\"$(b64 "redis://:$valkey_q@$valkey_service:6379/0")\",
\"oidc-client-secret\":\"$(b64 "$OIDC_CLIENT_SECRET")\",
\"aes-key\":\"$(b64 "$AES_KEY")\",
\"operations-token\":\"$(b64 "$OPERATIONS_TOKEN")\",
\"dex-config\":\"$(b64 "$dex_config")\"}}" | kubectl apply -f - >/dev/null

pull_secret_args=""
if command -v docker-credential-desktop >/dev/null 2>&1; then
  ghcr_credential="$(printf 'https://ghcr.io' | docker-credential-desktop get 2>/dev/null || true)"
  if [ -n "$ghcr_credential" ]; then
    ghcr_user="$(printf '%s' "$ghcr_credential" | jq -r '.Username // empty')"
    ghcr_token="$(printf '%s' "$ghcr_credential" | jq -r '.Secret // empty')"
    if [ -n "$ghcr_user" ] && [ -n "$ghcr_token" ]; then
      docker_auth="$(printf '%s:%s' "$ghcr_user" "$ghcr_token" | base64 | tr -d '\n')"
      docker_config="$(printf '{\"auths\":{\"ghcr.io\":{\"auth\":\"%s\"}}}' "$docker_auth")"
      printf '{\"apiVersion\":\"v1\",\"kind\":\"Secret\",\"metadata\":{\"name\":\"tenancit-ghcr\",\"namespace\":\"%s\"},\"type\":\"kubernetes.io/dockerconfigjson\",\"data\":{\".dockerconfigjson\":\"%s\"}}\n' \
        "$namespace" "$(b64 "$docker_config")" | kubectl apply -f - >/dev/null
      pull_secret_args="--set imagePullSecrets[0].name=tenancit-ghcr"
    fi
  fi
fi

# shellcheck disable=SC2086 -- optional Helm argument is intentionally split.
helm upgrade --install "$release" "$root_dir/deploy/helm/tenancit" \
  --namespace "$namespace" \
  --values "$values" \
  --set-string image.digest="$TENANCIT_IMAGE_DIGEST" \
  --set-string public.host="$host" \
  --set-string public.oidcHost="$oidc_host" \
  $pull_secret_args \
  --wait --timeout 10m

kubectl -n "$namespace" rollout restart deployment/"$release" deployment/"$release-audit-jobs" deployment/"$release-dex" >/dev/null
kubectl -n "$namespace" rollout status deployment/"$release" --timeout=5m
kubectl -n "$namespace" rollout status deployment/"$release-audit-jobs" --timeout=5m
kubectl -n "$namespace" rollout status deployment/"$release-dex" --timeout=5m
printf 'Tenancit: https://%s\nCredentials: %s (mode 0600)\n' "$host" "$state_file"
