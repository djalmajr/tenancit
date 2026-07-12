#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
digest="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT INT TERM

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
