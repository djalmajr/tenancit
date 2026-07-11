// Package auditops operates audit partitions, retention, legal holds, and exports.
package auditops

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PartitionHealth struct {
	CurrentMonthCovered bool      `json:"current_month_covered"`
	FutureThrough       time.Time `json:"future_through"`
	DefaultRows         int64     `json:"default_rows"`
	OldestRetained      time.Time `json:"oldest_retained,omitempty"`
	ActiveLegalHolds    int64     `json:"active_legal_holds"`
}

type RetentionResult struct {
	PartitionsCreated int `json:"partitions_created"`
	PartitionsDropped int `json:"partitions_dropped"`
	PartitionsHeld    int `json:"partitions_held"`
}

// Maintain delegates DDL to a fixed SECURITY DEFINER database function. The
// jobs role cannot create or drop arbitrary relations.
func Maintain(ctx context.Context, pool *pgxpool.Pool, now time.Time, retentionDays, futureMonths int) (RetentionResult, error) {
	var result RetentionResult
	if pool == nil || retentionDays < 1 || futureMonths < 1 || futureMonths > 24 {
		return result, fmt.Errorf("invalid audit maintenance configuration")
	}
	err := pool.QueryRow(ctx, `SELECT partitions_created,partitions_dropped,partitions_held FROM maintain_admin_audit_partitions($1,$2,$3)`, now.UTC(), retentionDays, futureMonths).Scan(&result.PartitionsCreated, &result.PartitionsDropped, &result.PartitionsHeld)
	if err != nil {
		return result, fmt.Errorf("maintain audit partitions: %w", err)
	}
	return result, nil
}

func Health(ctx context.Context, pool *pgxpool.Pool, now time.Time) (PartitionHealth, error) {
	var health PartitionHealth
	current := monthStart(now.UTC())
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_partition_registry WHERE from_time=$1),COALESCE((SELECT max(to_time) FROM audit_partition_registry),$1),(SELECT count(*) FROM admin_audit_events_default),COALESCE((SELECT min(from_time) FROM audit_partition_registry),$1),(SELECT count(*) FROM audit_legal_holds WHERE released_at IS NULL)`, current).Scan(&health.CurrentMonthCovered, &health.FutureThrough, &health.DefaultRows, &health.OldestRetained, &health.ActiveLegalHolds)
	if err != nil {
		return health, fmt.Errorf("load audit partition health: %w", err)
	}
	return health, nil
}

func monthStart(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
}
func partitionName(value time.Time) string {
	return "admin_audit_events_" + value.UTC().Format("200601")
}
