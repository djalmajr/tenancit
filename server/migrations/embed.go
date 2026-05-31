package migrations

import "embed"

// FS holds the goose SQL migrations embedded into the binary.
//
//go:embed *.sql
var FS embed.FS
