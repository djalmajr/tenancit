package webhook

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Delivery struct {
	ID, EventID, TargetID     uuid.UUID
	LeaseToken                uuid.UUID
	AttemptCount              int32
	ConsecutiveFailures       int32
	Format                    string
	URLCipher, URLNonce       []byte
	URLKeyVersion             int16
	SecretCipher, SecretNonce []byte
	SecretKeyVersion          int16
	AllowLoopbackHTTP         bool
	Event                     Event
}

type DeliveryStore struct{ pool *pgxpool.Pool }

func NewDeliveryStore(pool *pgxpool.Pool) *DeliveryStore { return &DeliveryStore{pool: pool} }

func (s *DeliveryStore) Claim(ctx context.Context, now time.Time, lease time.Duration, limit int32) ([]Delivery, error) {
	if s == nil || s.pool == nil || limit <= 0 {
		return nil, errors.New("delivery store and positive limit are required")
	}
	leaseToken := uuid.New()
	rows, err := s.pool.Query(ctx, `
		WITH candidates AS (
			SELECT d.id FROM webhook_deliveries d
			JOIN webhook_targets t ON t.id = d.target_id
			WHERE t.status = 'active'
			  AND (t.circuit_open_until IS NULL OR t.circuit_open_until <= $1)
			  AND ((d.status IN ('pending', 'retry') AND d.next_attempt_at <= $1)
			    OR (d.status = 'delivering' AND d.lease_expires_at <= $1))
			ORDER BY d.next_attempt_at, d.id
			FOR UPDATE OF d SKIP LOCKED LIMIT $2
		), claimed AS (
			UPDATE webhook_deliveries d SET status = 'delivering', attempt_count = attempt_count + 1,
				lease_token = $3, lease_expires_at = $4, updated_at = $1
			FROM candidates c WHERE d.id = c.id RETURNING d.*
		)
		SELECT c.id, c.event_id, c.target_id, c.lease_token, c.attempt_count,
			t.consecutive_failures, t.format, t.url_cipher, t.url_nonce, t.url_key_version,
			t.signing_secret_cipher, t.signing_secret_nonce, t.signing_secret_key_version,
			t.allow_loopback_http,
			e.id, e.event_type, e.event_version, e.aggregate_type, e.aggregate_id,
			e.request_id, e.occurred_at, e.payload
		FROM claimed c JOIN webhook_targets t ON t.id = c.target_id
		JOIN outbox_events e ON e.id = c.event_id ORDER BY c.id
	`, now.UTC(), limit, leaseToken, now.UTC().Add(lease))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deliveries := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.EventID, &d.TargetID, &d.LeaseToken, &d.AttemptCount,
			&d.ConsecutiveFailures, &d.Format, &d.URLCipher, &d.URLNonce, &d.URLKeyVersion,
			&d.SecretCipher, &d.SecretNonce, &d.SecretKeyVersion, &d.AllowLoopbackHTTP,
			&d.Event.ID, &d.Event.Type, &d.Event.Version, &d.Event.AggregateType, &d.Event.AggregateID,
			&d.Event.RequestID, &d.Event.OccurredAt, &d.Event.Payload); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}

func (s *DeliveryStore) MarkDelivered(ctx context.Context, d Delivery, now time.Time, status int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE webhook_deliveries SET status='delivered', delivered_at=$3,
		last_http_status=$4, last_error_code=NULL, lease_token=NULL, lease_expires_at=NULL, updated_at=$3
		WHERE id=$1 AND lease_token=$2 AND status='delivering'`, d.ID, d.LeaseToken, now.UTC(), status)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("webhook delivery lease lost")
	}
	if _, err = tx.Exec(ctx, `UPDATE webhook_targets SET consecutive_failures=0, circuit_open_until=NULL, updated_at=$2 WHERE id=$1`, d.TargetID, now.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *DeliveryStore) MarkFailed(ctx context.Context, d Delivery, now, next time.Time, status *int, code string, dead bool, openUntil *time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	state := "retry"
	if dead {
		state = "dead_letter"
	}
	command, err := tx.Exec(ctx, `UPDATE webhook_deliveries SET status=$3, next_attempt_at=$4,
		last_http_status=$5, last_error_code=$6, lease_token=NULL, lease_expires_at=NULL, updated_at=$7
		WHERE id=$1 AND lease_token=$2 AND status='delivering'`, d.ID, d.LeaseToken, state, next.UTC(), status, code, now.UTC())
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("webhook delivery lease lost")
	}
	if _, err = tx.Exec(ctx, `UPDATE webhook_targets SET consecutive_failures=consecutive_failures+1,
		circuit_open_until=$2, updated_at=$3 WHERE id=$1`, d.TargetID, openUntil, now.UTC()); err != nil {
		return err
	}
	if dead {
		if _, err = tx.Exec(ctx, `INSERT INTO webhook_dead_letters (delivery_id, reason_code) VALUES ($1,$2) ON CONFLICT (delivery_id) DO NOTHING`, d.ID, code); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *DeliveryStore) Replay(ctx context.Context, deliveryID uuid.UUID, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE webhook_deliveries SET status='pending', attempt_count=0,
		next_attempt_at=$2, last_error_code=NULL, last_http_status=NULL, updated_at=$2 WHERE id=$1 AND status='dead_letter'`, deliveryID, now.UTC())
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `UPDATE webhook_dead_letters SET replayed_at=$2 WHERE delivery_id=$1`, deliveryID, now.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
