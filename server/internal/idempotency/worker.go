package idempotency

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunCleanupWorker(ctx context.Context, pool *pgxpool.Pool, now func() time.Time, interval time.Duration) {
	if now == nil {
		now = time.Now
	}
	if interval <= 0 {
		interval = time.Hour
	}
	run := func() {
		started := time.Now()
		tx, err := pool.Begin(ctx)
		var removed int64
		if err == nil {
			removed, err = CleanupExpired(ctx, tx, now().UTC(), 1000)
			if err == nil {
				err = tx.Commit(ctx)
			} else {
				_ = tx.Rollback(ctx)
			}
		}
		outcome := "success"
		if err != nil {
			outcome = "error"
			slog.Error("cleanup admin idempotency", "error_type", fmt.Sprintf("%T", err))
		}
		telemetry.RecordWorkerCycle(ctx, "idempotency_cleanup", outcome, int(removed), time.Since(started))
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
