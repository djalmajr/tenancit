package settings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRevisionConflict = errors.New("settings revision conflict")
	ErrInvalidSettings  = errors.New("invalid settings")
)

type Actor struct {
	Kind, Issuer, Subject string
}

func (r *Repository) SessionPolicy(ctx context.Context) (time.Duration, time.Duration, error) {
	snapshot, err := r.Get(ctx)
	if err != nil {
		return 0, 0, err
	}
	absoluteHours, err := strconv.Atoi(snapshot.Values[SessionAbsoluteHours])
	if err != nil {
		return 0, 0, err
	}
	idleMinutes, err := strconv.Atoi(snapshot.Values[SessionIdleMinutes])
	if err != nil {
		return 0, 0, err
	}
	return time.Duration(absoluteHours) * time.Hour, time.Duration(idleMinutes) * time.Minute, nil
}

func (r *Repository) UsageRetentionMonths(ctx context.Context) (int, error) {
	snapshot, err := r.Get(ctx)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(snapshot.Values[UsageRetentionMonths])
}

func (r *Repository) AuditRetentionDays(ctx context.Context) (int, error) {
	snapshot, err := r.Get(ctx)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(snapshot.Values[AuditRetentionDays])
}

func (r *Repository) WebhookRetention(ctx context.Context) (int, int, error) {
	snapshot, err := r.Get(ctx)
	if err != nil {
		return 0, 0, err
	}
	deliveries, err := strconv.Atoi(snapshot.Values[WebhookDeliveryRetentionDays])
	if err != nil {
		return 0, 0, err
	}
	events, err := strconv.Atoi(snapshot.Values[OutboxEventRetentionDays])
	if err != nil {
		return 0, 0, err
	}
	return deliveries, events, nil
}

type Snapshot struct {
	Revision    int64             `json:"revision"`
	Values      map[string]string `json:"values"`
	Definitions []Definition      `json:"definitions"`
}

type Repository struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewRepository(pool *pgxpool.Pool, now func() time.Time) *Repository {
	if now == nil {
		now = time.Now
	}
	return &Repository{pool: pool, now: now}
}

func (r *Repository) Get(ctx context.Context) (Snapshot, error) {
	if r == nil || r.pool == nil {
		return Snapshot{}, errors.New("settings repository is unavailable")
	}
	return readSnapshot(ctx, r.pool)
}

func (r *Repository) Update(ctx context.Context, tx pgx.Tx, expectedRevision int64, updates map[string]string, actor Actor) (Snapshot, error) {
	if r == nil || tx == nil || expectedRevision <= 0 || actor.Kind == "" || actor.Subject == "" {
		return Snapshot{}, errors.New("complete settings update context is required")
	}
	var currentRevision int64
	if err := tx.QueryRow(ctx, `SELECT version FROM admin_settings_revision WHERE singleton = true FOR UPDATE`).Scan(&currentRevision); err != nil {
		return Snapshot{}, err
	}
	if currentRevision != expectedRevision {
		return Snapshot{}, ErrRevisionConflict
	}
	current, err := readValues(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	normalized, err := Validate(updates, current)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	updatedAt := r.now().UTC()
	for key, value := range normalized {
		_, err := tx.Exec(ctx, `
			INSERT INTO admin_settings (
				key, value, updated_at, updated_by_kind, updated_by_issuer, updated_by_subject
			) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
			ON CONFLICT (key) DO UPDATE SET
				value = EXCLUDED.value, updated_at = EXCLUDED.updated_at,
				updated_by_kind = EXCLUDED.updated_by_kind,
				updated_by_issuer = EXCLUDED.updated_by_issuer,
				updated_by_subject = EXCLUDED.updated_by_subject
		`, key, value, updatedAt, actor.Kind, actor.Issuer, actor.Subject)
		if err != nil {
			return Snapshot{}, err
		}
		current[key] = value
	}
	nextRevision := currentRevision + 1
	command, err := tx.Exec(ctx, `UPDATE admin_settings_revision SET version = $1, updated_at = $2 WHERE singleton = true`, nextRevision, updatedAt)
	if err != nil {
		return Snapshot{}, err
	}
	if command.RowsAffected() != 1 {
		return Snapshot{}, fmt.Errorf("settings revision row is missing")
	}
	values := Defaults()
	for key, value := range current {
		values[key] = value
	}
	return Snapshot{Revision: nextRevision, Values: values, Definitions: Definitions()}, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readSnapshot(ctx context.Context, q queryer) (Snapshot, error) {
	var revision int64
	if err := q.QueryRow(ctx, `SELECT version FROM admin_settings_revision WHERE singleton = true`).Scan(&revision); err != nil {
		return Snapshot{}, err
	}
	stored, err := readValues(ctx, q)
	if err != nil {
		return Snapshot{}, err
	}
	values := Defaults()
	for key, value := range stored {
		values[key] = value
	}
	return Snapshot{Revision: revision, Values: values, Definitions: Definitions()}, nil
}

func readValues(ctx context.Context, q queryer) (map[string]string, error) {
	rows, err := q.Query(ctx, `SELECT key, value FROM admin_settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}
