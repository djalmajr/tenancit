#!/bin/sh
set -eu

ref_type="${TENANCIT_IMAGE_REF_TYPE:-}"
ref_name="${TENANCIT_IMAGE_REF_NAME:-}"
publish_latest=false

# Only an exact stable SemVer tag may move the mutable latest alias. Any
# prerelease or manually dispatched branch build remains addressable by its
# immutable SHA (and, for tag events, by the version tag).
if [ "$ref_type" = "tag" ] &&
  printf '%s\n' "$ref_name" |
    LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  publish_latest=true
fi

output="publish_latest=$publish_latest"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  printf '%s\n' "$output" >>"$GITHUB_OUTPUT"
else
  printf '%s\n' "$output"
fi
