package usage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type RetentionStore interface {
	DeleteExpiredAPIClientUsage(context.Context, pgtype.Date) (int64, error)
}

type RetentionPolicy interface {
	UsageRetentionMonths(context.Context) (int, error)
}

func RetentionCutoff(now time.Time) time.Time {
	return retentionCutoffMonths(now, 6)
}

func retentionCutoffMonths(now time.Time, months int) time.Time {
	cutoff := now.UTC().AddDate(0, -months, 0)
	year, month, day := cutoff.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func RunRetention(ctx context.Context, store RetentionStore, policy RetentionPolicy, now func() time.Time, interval time.Duration) {
	run := func() {
		months := 6
		if policy != nil {
			configured, err := policy.UsageRetentionMonths(ctx)
			if err != nil || configured <= 0 {
				slog.Error("load API client usage retention policy", "error_type", fmt.Sprintf("%T", err))
				return
			}
			months = configured
		}
		cutoff := retentionCutoffMonths(now(), months)
		deleted, err := store.DeleteExpiredAPIClientUsage(ctx, pgtype.Date{Time: cutoff, Valid: true})
		if err != nil {
			slog.Error("delete expired API client usage", "err", err)
			return
		}
		slog.Info("API client usage retention complete", "deleted", deleted, "cutoff", cutoff.Format("2006-01-02"))
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

var _ RetentionStore = (*db.Queries)(nil)
