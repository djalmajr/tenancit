#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
unformatted="$(find "$root_dir/server" -type f -name '*.go' -print0 | xargs -0 gofmt -l)"
if [ -n "$unformatted" ]; then
  echo "gofmt is required for:" >&2
  echo "$unformatted" >&2
  exit 1
fi

cd "$root_dir/server"
if go list -deps ./cmd/server | grep -qx 'github.com/djalmajr/tenancit/server/internal/migration'; then
  echo "HTTP runtime must not import the migration package" >&2
  exit 1
fi
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
