package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RetentionPolicy interface {
	WebhookRetention(context.Context) (int, int, error)
}

func RunRetention(ctx context.Context, pool *pgxpool.Pool, policy RetentionPolicy, now func() time.Time, interval time.Duration) {
	run := func() {
		started := time.Now()
		deliveryDays, eventDays, err := policy.WebhookRetention(ctx)
		if err != nil || deliveryDays <= 0 || eventDays <= 0 {
			telemetry.RecordWorkerCycle(ctx, "webhook_retention", "error", 0, time.Since(started))
			slog.Error("load webhook retention policy", "error_type", fmt.Sprintf("%T", err))
			return
		}
		if err := runRetentionOnce(ctx, pool, now().UTC(), deliveryDays, eventDays); err != nil {
			telemetry.RecordDependencyOperation(ctx, "outbox", "retain", "error", time.Since(started))
			telemetry.RecordWorkerCycle(ctx, "webhook_retention", "error", 0, time.Since(started))
			slog.Error("apply webhook retention", "error_type", fmt.Sprintf("%T", err))
			return
		}
		telemetry.RecordDependencyOperation(ctx, "outbox", "retain", "success", time.Since(started))
		telemetry.RecordWorkerCycle(ctx, "webhook_retention", "success", 0, time.Since(started))
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

func runRetentionOnce(ctx context.Context, pool *pgxpool.Pool, current time.Time, deliveryDays, eventDays int) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin retention: %w", err)
	}
	defer tx.Rollback(ctx)
	deliveryCutoff := current.UTC().AddDate(0, 0, -deliveryDays)
	eventCutoff := current.UTC().AddDate(0, 0, -eventDays)
	if _, err = tx.Exec(ctx, `DELETE FROM webhook_dead_letters dl USING webhook_deliveries d WHERE dl.delivery_id=d.id AND d.status IN ('delivered','dead_letter') AND d.updated_at<$1`, deliveryCutoff); err != nil {
		return fmt.Errorf("delete dead letters: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM webhook_deliveries WHERE status IN ('delivered','dead_letter') AND updated_at<$1`, deliveryCutoff); err != nil {
		return fmt.Errorf("delete deliveries: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM outbox_events e WHERE e.occurred_at<$1 AND NOT EXISTS(SELECT 1 FROM webhook_deliveries d WHERE d.event_id=e.id)`, eventCutoff); err != nil {
		return fmt.Errorf("delete orphan events: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit retention: %w", err)
	}
	return nil
}
