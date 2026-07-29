#!/bin/sh
set -eu

root_dir="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
classifier="$root_dir/scripts/classify-image-release.sh"
workflow="$root_dir/.github/workflows/publish-image.yml"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

fail() {
  echo "publish-image contract failed: $*" >&2
  exit 1
}

assert_channel() {
  expected="$1"
  ref_type="$2"
  ref_name="$3"
  actual="$(
    TENANCIT_IMAGE_REF_TYPE="$ref_type" \
      TENANCIT_IMAGE_REF_NAME="$ref_name" \
      "$classifier"
  )"
  [ "$actual" = "publish_latest=$expected" ] ||
    fail "$ref_type $ref_name produced '$actual'; expected publish_latest=$expected"
}

[ -x "$classifier" ] || fail "missing executable classifier: $classifier"

assert_channel true tag v1.2.3
assert_channel true tag v0.0.0
assert_channel false tag v1.2.3-alpha.1
assert_channel false tag v1.2.3-beta.2
assert_channel false tag v1.2.3-rc.3
assert_channel false tag v1.2.3-preview.1
assert_channel false tag v1.2
assert_channel false branch main

GITHUB_OUTPUT="$tmpdir/github-output" \
  TENANCIT_IMAGE_REF_TYPE=tag \
  TENANCIT_IMAGE_REF_NAME=v2.0.0 \
  "$classifier" >"$tmpdir/stdout"
[ ! -s "$tmpdir/stdout" ] || fail "classifier wrote workflow output to stdout"
grep -Fxq "publish_latest=true" "$tmpdir/github-output" ||
  fail "classifier did not write publish_latest to GITHUB_OUTPUT"

grep -Fq "sh ./scripts/classify-image-release.sh" "$workflow" ||
  fail "workflow does not execute the release-channel classifier"
grep -Fq "flavor: latest=false" "$workflow" ||
  fail "metadata-action automatic latest flavor is not disabled"
# GitHub expression must remain literal.
# shellcheck disable=SC2016
grep -Fq 'type=raw,value=latest,enable=${{ steps.release_channel.outputs.publish_latest }}' "$workflow" ||
  fail "workflow does not gate latest on the classifier output"

echo "publish-image workflow contract tests passed"
