package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/rewrap"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type options struct {
	rewrap.Config
	DatabaseURL  string
	ReportURL    string
	ReportToken  string
	ReportSource string
	DevMode      bool
}

func main() {
	slog.SetDefault(slog.New(telemetry.NewRedactingJSONHandler(os.Stdout, nil)))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := execute(ctx, os.Getenv, os.Args[1:], os.Stdout); err != nil {
		slog.Error("rewrap failed", "error_type", fmt.Sprintf("%T", err), "reason", publicReason(err))
		os.Exit(1)
	}
}

func execute(ctx context.Context, getenv func(string) string, args []string, output io.Writer) error {
	options, err := parseOptions(getenv, args)
	if err != nil {
		return err
	}
	cryptor, err := crypto.FromEnv()
	if err != nil {
		return err
	}
	telemetryConfig, err := telemetry.LoadRuntimeConfig(getenv)
	if err != nil {
		return err
	}
	if telemetryConfig.ServiceName == "tenancit" {
		telemetryConfig.ServiceName = "tenancit-rewrap"
	}
	shutdownTelemetry, err := telemetry.SetupRuntime(ctx, telemetryConfig)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTelemetry(shutdownContext)
	}()
	poolConfig, err := pgxpool.ParseConfig(options.DatabaseURL)
	if err != nil {
		return errors.New("invalid rewrap database URL")
	}
	poolConfig.ConnConfig.Tracer = telemetry.NewPGXTracer()
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return errors.New("connect rewrap database")
	}
	defer pool.Close()

	var reporter rewrap.Reporter
	if !options.DryRun {
		reporter, err = rewrap.NewHTTPReporter(options.ReportURL, options.ReportToken, options.ReportSource, options.DevMode, &http.Client{Timeout: 10 * time.Second})
		if err != nil {
			return err
		}
	}
	runner := &rewrap.Runner{DB: pool, Cryptor: cryptor, Reporter: reporter}
	summary, err := runner.Run(ctx, options.Config)
	if err != nil {
		if reporter != nil && summary.JobID != "" {
			reportContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = reporter.Report(reportContext, summary, "failed")
			cancel()
		}
		return err
	}
	return json.NewEncoder(output).Encode(map[string]any{
		"job_id": summary.JobID, "target_version": summary.TargetVersion,
		"dry_run": summary.DryRun, "rows_scanned": summary.Scanned,
		"rows_rewrapped": summary.Rewrapped, "batches": summary.Batches,
		"locked_retries": summary.LockedRetries, "rows_remaining": summary.Remaining,
		"inventory":  summary.Inventory,
		"started_at": summary.StartedAt, "completed_at": summary.CompletedAt,
	})
}

func parseOptions(getenv func(string) string, args []string) (options, error) {
	parsed := options{
		DatabaseURL:  strings.TrimSpace(getenv("TENANCIT_REWRAP_DATABASE_URL")),
		ReportURL:    strings.TrimSpace(getenv("TENANCIT_OPERATIONS_BASE_URL")),
		ReportToken:  strings.TrimSpace(getenv("TENANCIT_OPERATIONS_REPORT_TOKEN")),
		ReportSource: strings.TrimSpace(getenv("TENANCIT_REWRAP_SOURCE")),
		DevMode:      strings.EqualFold(strings.TrimSpace(getenv("TENANCIT_DEV_MODE")), "true"),
	}
	flags := flag.NewFlagSet("tenancit-rewrap", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.IntVar(&parsed.TargetVersion, "target-version", 0, "current target key version")
	flags.IntVar(&parsed.BatchSize, "batch-size", 100, "rows per transaction (1-1000)")
	flags.BoolVar(&parsed.DryRun, "dry-run", false, "authenticate every ciphertext without writes")
	flags.BoolVar(&parsed.ConfirmedWrite, "confirm-write", false, "explicitly authorize rewrap writes")
	flags.DurationVar(&parsed.MaxDuration, "max-duration", time.Hour, "campaign deadline")
	flags.DurationVar(&parsed.NoProgressTimeout, "no-progress-timeout", 30*time.Second, "locked-row progress deadline")
	flags.StringVar(&parsed.JobID, "job-id", "", "change UUID")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return options{}, errors.New("invalid rewrap arguments")
	}
	if parsed.DatabaseURL == "" {
		return options{}, errors.New("TENANCIT_REWRAP_DATABASE_URL is required")
	}
	if parsed.TargetVersion <= 0 || parsed.BatchSize < 1 || parsed.BatchSize > 1000 || parsed.MaxDuration <= 0 || parsed.NoProgressTimeout <= 0 || (parsed.DryRun && parsed.ConfirmedWrite) {
		return options{}, errors.New("invalid rewrap configuration")
	}
	if !parsed.DryRun && parsed.JobID == "" {
		return options{}, errors.New("write mode requires an approved job UUID")
	}
	if parsed.JobID != "" {
		if _, err := uuid.Parse(parsed.JobID); err != nil {
			return options{}, errors.New("job ID must be a UUID")
		}
	}
	if !parsed.DryRun && (parsed.ReportURL == "" || parsed.ReportToken == "" || parsed.ReportSource == "") {
		return options{}, errors.New("write mode requires operations report URL, token, and source")
	}
	return parsed, nil
}

func publicReason(err error) string {
	known := []struct {
		err    error
		reason string
	}{
		{rewrap.ErrCampaignLocked, "campaign_locked"},
		{rewrap.ErrInvalidConfig, "invalid_config"},
		{rewrap.ErrSafetyEvidence, "safety_evidence_missing"},
		{rewrap.ErrMalformedCiphertext, "malformed_ciphertext"},
		{rewrap.ErrMissingKeyVersion, "key_version_unavailable"},
		{rewrap.ErrAuthentication, "authentication_failed"},
		{rewrap.ErrVerification, "verification_failed"},
		{rewrap.ErrCASConflict, "cas_conflict"},
		{rewrap.ErrNoProgress, "no_progress"},
	}
	for _, item := range known {
		if errors.Is(err, item.err) {
			return item.reason
		}
	}
	return "internal_error"
}
