package db

import (
	"context"

	"github.com/google/uuid"
)

type InsertOutboxEventParams struct {
	EventType     string
	EventVersion  int32
	AggregateType string
	AggregateID   string
	RequestID     string
	Payload       []byte
}

type InsertedOutboxEvent struct{ ID uuid.UUID }

func (q *Queries) InsertOutboxEvent(ctx context.Context, arg InsertOutboxEventParams) (InsertedOutboxEvent, error) {
	const statement = `
		INSERT INTO outbox_events (
			event_type, event_version, aggregate_type, aggregate_id, request_id, payload
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var event InsertedOutboxEvent
	err := q.db.QueryRow(ctx, statement,
		arg.EventType, arg.EventVersion, arg.AggregateType, arg.AggregateID, arg.RequestID, arg.Payload,
	).Scan(&event.ID)
	return event, err
}

func (q *Queries) EnqueueOutboxDeliveries(ctx context.Context, eventID uuid.UUID) error {
	const statement = `
		INSERT INTO webhook_deliveries (event_id, target_id)
		SELECT $1, id
		FROM webhook_targets
		WHERE status = 'active'
		  AND (circuit_open_until IS NULL OR circuit_open_until <= clock_timestamp())
		ON CONFLICT (event_id, target_id) DO NOTHING
	`
	_, err := q.db.Exec(ctx, statement, eventID)
	return err
}
