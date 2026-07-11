package usage

import (
	"context"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type RetentionStore interface {
	DeleteExpiredAPIClientUsage(context.Context, pgtype.Date) (int64, error)
}

func RetentionCutoff(now time.Time) time.Time {
	cutoff := now.UTC().AddDate(0, -6, 0)
	year, month, day := cutoff.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func RunRetention(ctx context.Context, store RetentionStore, now func() time.Time, interval time.Duration) {
	run := func() {
		cutoff := RetentionCutoff(now())
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
