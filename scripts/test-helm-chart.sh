#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
digest="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
tmp="$(mktemp)"
shared_tmp="$(mktemp)"
trap 'rm -f "$tmp" "$shared_tmp"' EXIT INT TERM

if command -v helm >/dev/null 2>&1; then
  helm_cmd=helm
else
  command -v docker >/dev/null 2>&1 || { echo "helm or docker is required" >&2; exit 1; }
  helm_cmd="docker run --rm -v $root_dir:/work -w /work alpine/helm:4.2.2"
fi

# shellcheck disable=SC2086 -- helm_cmd may intentionally contain docker arguments.
$helm_cmd lint deploy/helm/tenancit \
  --values deploy/helm/tenancit/values-personal.yaml \
  --set-string image.digest="$digest"
# shellcheck disable=SC2086
$helm_cmd template tenancit deploy/helm/tenancit \
  --namespace tenancit \
  --values deploy/helm/tenancit/values-personal.yaml \
  --set-string image.digest="$digest" >"$tmp"

grep -q "ghcr.io/djalmajr/tenancit@$digest" "$tmp"
grep -q 'replicas: 2' "$tmp"
grep -q 'TENANCIT_ADMIN_AUTH_MODE' "$tmp"
grep -q 'kind: PodDisruptionBudget' "$tmp"
grep -q 'kind: StatefulSet' "$tmp"
grep -q 'kind: NetworkPolicy' "$tmp"

# Shared-host contract: only the prefixed human-admin surface is public. The
# application serves its SPA and admin/auth APIs below the same base path;
# root /v1 APIs, operations endpoints and health probes stay ClusterIP-only.
# shellcheck disable=SC2086
$helm_cmd template tenancit deploy/helm/tenancit \
  --namespace hyper \
  --set-string image.digest="$digest" \
  --set-string public.host=admin-labdev.cloud4biz.com \
  --set-string public.oidcHost=auth-labdev.cloud4biz.com \
  --set-string ingress.className=nginx \
  --set-string ingress.tlsSecretName=cloud4biz-tls \
  --set-string ingress.path=/tenancit \
  --set-string app.basePath=/tenancit \
  --set-string oidc.issuerPath=/realms/admin-labdev >"$shared_tmp"

grep -Eq -- '^          - path: "?/tenancit"?$' "$shared_tmp"
if [ "$(grep -c -- '^          - path:' "$shared_tmp")" -ne 1 ]; then
  echo "shared ingress must expose exactly one path: /tenancit" >&2
  exit 1
fi
grep -q 'name: TENANCIT_BASE_PATH' "$shared_tmp"
grep -q 'value: "/tenancit"' "$shared_tmp"
grep -q 'value: "https://admin-labdev.cloud4biz.com"' "$shared_tmp"
grep -q 'value: "https://auth-labdev.cloud4biz.com/realms/admin-labdev"' "$shared_tmp"
grep -q 'path: /readyz' "$shared_tmp"
grep -q 'path: /healthz' "$shared_tmp"

if $helm_cmd template tenancit deploy/helm/tenancit \
  --set-string image.digest="$digest" \
  --set ingress.enabled=true \
  --set-string adminAuth.mode=legacy_shared_token >/dev/null 2>&1; then
  echo "public ingress accepted legacy shared-token admin auth" >&2
  exit 1
fi

if $helm_cmd template tenancit deploy/helm/tenancit \
  --set-string image.digest="$digest" \
  --set-string ingress.path=/tenancit \
  --set-string app.basePath=/other >/dev/null 2>&1; then
  echo "mismatched ingress and application base paths were accepted" >&2
  exit 1
fi

for invalid in latest sha256:abc sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA; do
  if $helm_cmd template tenancit deploy/helm/tenancit \
    --values deploy/helm/tenancit/values-personal.yaml \
    --set-string image.digest="$invalid" >/dev/null 2>&1; then
    echo "invalid image digest was accepted: $invalid" >&2
    exit 1
  fi
done

for key in personal.postgresImage personal.valkeyImage personal.dexImage; do
  if $helm_cmd template tenancit deploy/helm/tenancit \
    --values deploy/helm/tenancit/values-personal.yaml \
    --set-string image.digest="$digest" --set-string "$key=example:latest" >/dev/null 2>&1; then
    echo "mutable personal image was accepted: $key" >&2
    exit 1
  fi
done
