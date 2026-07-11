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

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/httpapi"
	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	"github.com/djalmajr/tenancit/server/internal/spa"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/usage"
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
	authConfig, err := adminauth.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}

	if err := store.Migrate(dsn); err != nil {
		return err
	}
	cryptor, err := crypto.FromEnv()
	if err != nil {
		return err
	}

	ctx, cancelRuntime := context.WithCancel(context.Background())
	defer cancelRuntime()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	staticHandler, err := spa.Handler()
	if err != nil {
		return err
	}

	srv := httpapi.NewServer(pool, cryptor, authConfig.LegacyToken)
	authStore := adminauth.NewPostgresSessionStore(pool)
	srv.SetAdminAuthStore(authStore)
	srv.ConfigureAdminAuth(authConfig, nil, nil)
	if authConfig.Mode == adminauth.ModeOIDC {
		discoveryCtx, cancelDiscovery := context.WithTimeout(ctx, 10*time.Second)
		provider, err := adminauth.NewProvider(discoveryCtx, authConfig.OIDC)
		cancelDiscovery()
		if err != nil {
			return err
		}
		sessions := adminauth.NewSessionManager(
			authStore, cryptor, nil, time.Now,
			authConfig.SessionAbsolute, authConfig.SessionIdle,
		)
		sessions.SetPolicyProvider(srv.Settings)
		oidcManager := adminauth.NewOIDCManager(authConfig.OIDC, provider, authStore, sessions, cryptor, nil, time.Now)
		srv.ConfigureAdminAuth(authConfig, oidcManager, sessions)
	}
	limiterMode := envOr("TENANCIT_RATE_LIMIT_MODE", "valkey")
	if limiterMode == "memory" {
		slog.Warn("using single-instance in-memory rate limiter")
		srv.SetRateLimiter(ratelimit.NewMemory(time.Now))
	} else {
		valkeyURL := os.Getenv("TENANCIT_VALKEY_URL")
		if valkeyURL == "" {
			return errors.New("TENANCIT_VALKEY_URL is required unless TENANCIT_RATE_LIMIT_MODE=memory")
		}
		valkeyLimiter, err := ratelimit.NewValkey(valkeyURL)
		if err != nil {
			return err
		}
		defer valkeyLimiter.Close()
		pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
		defer cancelPing()
		if err := valkeyLimiter.Ping(pingCtx); err != nil {
			return errors.Join(ratelimit.ErrUnavailable, err)
		}
		srv.SetRateLimiter(valkeyLimiter)
	}
	usageCollector := usage.NewCollector(db.New(pool), 4096, 10*time.Second)
	srv.SetUsageRecorder(usageCollector)
	go usageCollector.Run(ctx)
	go usage.RunRetention(ctx, db.New(pool), srv.Settings, time.Now, 24*time.Hour)
	httpServer := newHTTPServer(addr, srv.Routes(staticHandler))

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
	err = httpServer.Shutdown(shutCtx)
	cancelRuntime()
	return err
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
