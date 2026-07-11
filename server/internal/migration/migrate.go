// Package migration owns schema evolution and database privilege reconciliation.
// The HTTP runtime deliberately does not import this package.
package migration

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/djalmajr/tenancit/server/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed runtime_grants.sql
var runtimeGrantsFS embed.FS

// Up applies every pending expand-compatible migration.
func Up(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// ApplyRuntimeGrants reconciles fixed group roles after every migration.
// Login roles are provisioned externally and receive membership in one group.
func ApplyRuntimeGrants(dsn string) error {
	grants, err := runtimeGrantsFS.ReadFile("runtime_grants.sql")
	if err != nil {
		return fmt.Errorf("read runtime grants: %w", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db for runtime grants: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(string(grants)); err != nil {
		return fmt.Errorf("apply runtime grants: %w", err)
	}
	return nil
}
