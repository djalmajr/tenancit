package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/djalmajr/tenancit/server/internal/auditops"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/idempotency"
	appsettings "github.com/djalmajr/tenancit/server/internal/settings"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(telemetry.NewRedactingJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "error_type", fmt.Sprintf("%T", err))
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("TENANCIT_JOBS_DATABASE_URL")
	if dsn == "" {
		return errors.New("TENANCIT_JOBS_DATABASE_URL is required")
	}
	cryptor, err := crypto.FromEnv()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	telemetryConfig, err := telemetry.LoadRuntimeConfig(os.Getenv)
	if err != nil {
		return err
	}
	shutdown, err := telemetry.SetupRuntime(ctx, telemetryConfig)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		_ = shutdown(shutdownCtx)
	}()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return err
	}
	poolConfig.ConnConfig.Tracer = telemetry.NewPGXTracer()
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	repository := auditops.NewExportRepository(pool, cryptor, time.Now)
	settings := appsettings.NewRepository(pool, time.Now)
	go auditops.RunExportWorker(ctx, repository, time.Second)
	go auditops.RunMaintenance(ctx, pool, settings, time.Now, 24*time.Hour)
	go idempotency.RunCleanupWorker(ctx, pool, time.Now, time.Hour)
	<-ctx.Done()
	return nil
}
