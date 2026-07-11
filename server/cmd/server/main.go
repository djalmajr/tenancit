package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/httpapi"
	"github.com/djalmajr/tenancit/server/internal/ratelimit"
	"github.com/djalmajr/tenancit/server/internal/spa"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/djalmajr/tenancit/server/internal/usage"
	"github.com/djalmajr/tenancit/server/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(telemetry.NewRedactingJSONHandler(os.Stdout, nil)))
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

	cryptor, err := crypto.FromEnv()
	if err != nil {
		return err
	}

	ctx, cancelRuntime := context.WithCancel(context.Background())
	defer cancelRuntime()
	telemetryConfig, err := telemetry.LoadRuntimeConfig(os.Getenv)
	if err != nil {
		return err
	}
	shutdownTelemetry, err := telemetry.SetupRuntime(ctx, telemetryConfig)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownContext); err != nil {
			slog.Error("shutdown telemetry", "error_type", fmt.Sprintf("%T", err))
		}
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

	staticHandler, err := spa.Handler()
	if err != nil {
		return err
	}

	srv := httpapi.NewServer(pool, cryptor, authConfig.LegacyToken)
	operationsToken, operationsVersion, err := loadOperationsCredential(os.Getenv)
	if err != nil {
		return err
	}
	srv.SetOperationsReportCredential(operationsToken, operationsVersion)
	readinessProbes := []telemetry.Probe{
		telemetry.NewPingProbe("postgres", true, pool.Ping),
	}
	allowWebhookLoopback := authConfig.DevMode && strings.EqualFold(strings.TrimSpace(os.Getenv("TENANCIT_WEBHOOK_ALLOW_LOOPBACK_HTTP")), "true")
	srv.SetWebhookTargets(webhook.NewTargetRepository(pool, cryptor, nil, nil, allowWebhookLoopback))
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
		readinessProbes = append(readinessProbes, telemetry.NewHTTPProbe(
			"oidc", authConfig.OIDC.Issuer+"/.well-known/openid-configuration", false, nil,
		))
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
		readinessProbes = append(readinessProbes, telemetry.NewPingProbe("valkey", true, valkeyLimiter.Ping))
	}
	srv.ReadinessProbes = readinessProbes
	usageCollector := usage.NewCollector(db.New(pool), 4096, 10*time.Second)
	srv.SetUsageRecorder(usageCollector)
	go usageCollector.Run(ctx)
	go usage.RunRetention(ctx, db.New(pool), srv.Settings, time.Now, 24*time.Hour)
	webhookWorker := webhook.NewWorker(webhook.NewDeliveryStore(pool), cryptor)
	go webhookWorker.Run(ctx, time.Second)
	go webhook.RunRetention(ctx, pool, srv.Settings, time.Now, 24*time.Hour)
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

func loadOperationsCredential(getenv func(string) string) (string, string, error) {
	token := strings.TrimSpace(getenv("TENANCIT_OPERATIONS_REPORT_TOKEN"))
	version := strings.TrimSpace(getenv("TENANCIT_OPERATIONS_REPORT_CREDENTIAL_VERSION"))
	if token == "" && version == "" {
		return "", "", nil
	}
	if len(token) < 32 || version == "" {
		return "", "", errors.New("operations report credential requires a token with at least 32 characters and a version")
	}
	return token, version, nil
}
