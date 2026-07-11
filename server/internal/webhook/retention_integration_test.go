package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRetentionDeletesOnlyExpiredTerminalRecordsAndOrphanEvents(t *testing.T) {
	ctx := context.Background()
	pool := testsupport.NewDB(t)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	oldDelivery := now.AddDate(0, 0, -31)
	oldEvent := now.AddDate(0, 0, -91)

	var targetID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO webhook_targets
		(name,format,status,url_cipher,url_nonce,url_key_version,signing_secret_cipher,signing_secret_nonce,signing_secret_key_version)
		VALUES ('retention','generic','active','x','x',1,'x','x',1) RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("insert target: %v", err)
	}
	var attachedEventID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO outbox_events
		(event_type,event_version,aggregate_type,aggregate_id,request_id,payload,occurred_at)
		VALUES ('tenancit.tenant.created',1,'tenant','attached','request-attached','{}',$1) RETURNING id`, oldEvent).Scan(&attachedEventID); err != nil {
		t.Fatalf("insert attached event: %v", err)
	}
	var deliveryID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO webhook_deliveries
		(event_id,target_id,status,updated_at,delivered_at) VALUES ($1,$2,'dead_letter',$3,$3) RETURNING id`, attachedEventID, targetID, oldDelivery).Scan(&deliveryID); err != nil {
		t.Fatalf("insert delivery: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO webhook_dead_letters (delivery_id,reason_code,failed_at) VALUES ($1,'permanent_http_status',$2)`, deliveryID, oldDelivery); err != nil {
		t.Fatalf("insert dead letter: %v", err)
	}
	var orphanEventID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO outbox_events
		(event_type,event_version,aggregate_type,aggregate_id,request_id,payload,occurred_at)
		VALUES ('tenancit.tenant.deleted',1,'tenant','orphan','request-orphan','{}',$1) RETURNING id`, oldEvent).Scan(&orphanEventID); err != nil {
		t.Fatalf("insert orphan event: %v", err)
	}
	var recentEventID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO outbox_events
		(event_type,event_version,aggregate_type,aggregate_id,request_id,payload,occurred_at)
		VALUES ('tenancit.tenant.created',1,'tenant','recent','request-recent','{}',$1) RETURNING id`, now).Scan(&recentEventID); err != nil {
		t.Fatalf("insert recent event: %v", err)
	}

	if err := runRetentionOnce(ctx, pool, now, 30, 90); err != nil {
		t.Fatalf("run retention: %v", err)
	}

	assertRowCount(t, pool, `SELECT count(*) FROM webhook_dead_letters WHERE delivery_id=$1`, deliveryID, 0)
	assertRowCount(t, pool, `SELECT count(*) FROM webhook_deliveries WHERE id=$1`, deliveryID, 0)
	assertRowCount(t, pool, `SELECT count(*) FROM outbox_events WHERE id=$1`, attachedEventID, 0)
	assertRowCount(t, pool, `SELECT count(*) FROM outbox_events WHERE id=$1`, orphanEventID, 0)
	assertRowCount(t, pool, `SELECT count(*) FROM outbox_events WHERE id=$1`, recentEventID, 1)
}

func assertRowCount(t *testing.T, pool *pgxpool.Pool, query string, id uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), query, id).Scan(&got); err != nil {
		t.Fatalf("count row: %v", err)
	}
	if got != want {
		t.Fatalf("count=%d want=%d query=%s", got, want, query)
	}
}
