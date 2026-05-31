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

	"github.com/centralit/resource-tenant/server/internal/crypto"
	"github.com/centralit/resource-tenant/server/internal/httpapi"
	"github.com/centralit/resource-tenant/server/internal/spa"
	"github.com/centralit/resource-tenant/server/internal/store"
	"github.com/centralit/resource-tenant/server/internal/store/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := envOr("RT_ADDR", ":8080")
	dsn := os.Getenv("RT_DATABASE_URL")
	if dsn == "" {
		return errors.New("RT_DATABASE_URL is required")
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

	srv := httpapi.NewServer(db.New(pool), cryptor)
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
