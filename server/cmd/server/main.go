package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/httpapi"
	"github.com/djalmajr/tenancit/server/internal/spa"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := envOr("TENANCIT_ADDR", ":8080")
	dsn := os.Getenv("TENANCIT_DATABASE_URL")
	if dsn == "" {
		return errors.New("TENANCIT_DATABASE_URL is required")
	}
	adminToken := os.Getenv("TENANCIT_ADMIN_TOKEN")
	if adminToken == "" {
		return errors.New("TENANCIT_ADMIN_TOKEN is required")
	}

	if err := store.Migrate(dsn); err != nil {
		return err
	}
	cryptor, err := crypto.FromEnv()
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	staticHandler, err := spa.Handler()
	if err != nil {
		return err
	}

	srv := httpapi.NewServer(pool, cryptor, adminToken)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(staticHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutCtx)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
