package auditops

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RetentionPolicy interface {
	AuditRetentionDays(context.Context) (int, error)
}

func RunExportWorker(ctx context.Context, repository *ExportRepository, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	run := func() {
		started := time.Now()
		items := 0
		outcome := "success"
		for {
			processed, err := repository.ProcessPending(ctx)
			if err != nil {
				outcome = "error"
				slog.Error("process audit export", "error_type", fmt.Sprintf("%T", err))
				break
			}
			if !processed {
				break
			}
			items++
		}
		if _, err := repository.Expire(ctx); err != nil {
			outcome = "error"
			slog.Error("expire audit exports", "error_type", fmt.Sprintf("%T", err))
		}
		telemetry.RecordWorkerCycle(ctx, "audit_export", outcome, items, time.Since(started))
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func RunMaintenance(ctx context.Context, pool *pgxpool.Pool, policy RetentionPolicy, now func() time.Time, interval time.Duration) {
	if now == nil {
		now = time.Now
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	run := func() {
		started := time.Now()
		days, err := policy.AuditRetentionDays(ctx)
		if err != nil || days < 1 {
			telemetry.RecordWorkerCycle(ctx, "audit_retention", "error", 0, time.Since(started))
			slog.Error("load audit retention policy", "error_type", fmt.Sprintf("%T", err))
			return
		}
		result, err := Maintain(ctx, pool, now().UTC(), days, 3)
		outcome := "success"
		if err != nil {
			outcome = "error"
			slog.Error("maintain audit partitions", "error_type", fmt.Sprintf("%T", err))
		}
		telemetry.RecordWorkerCycle(ctx, "audit_retention", outcome, result.PartitionsCreated+result.PartitionsDropped, time.Since(started))
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
