package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/djalmajr/tenancit/server/internal/migration"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
)

func main() {
	slog.SetDefault(slog.New(telemetry.NewRedactingJSONHandler(os.Stdout, nil)))
	if err := run(os.Getenv); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migration complete")
}

func run(getenv func(string) string) error {
	dsn := getenv("TENANCIT_DATABASE_URL")
	if dsn == "" {
		return errors.New("TENANCIT_DATABASE_URL is required")
	}
	if err := migration.Up(dsn); err != nil {
		return err
	}
	return migration.ApplyRuntimeGrants(dsn)
}
