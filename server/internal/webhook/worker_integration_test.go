package webhook

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
)

func webhookCryptor(t *testing.T) *appcrypto.Cryptor {
	t.Helper()
	c, err := appcrypto.New(map[int][]byte{1: bytes.Repeat([]byte{0x66}, 32)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func seedDelivery(t *testing.T, endpoint string) (*DeliveryStore, *appcrypto.Cryptor, CreatedTarget, uuid.UUID) {
	t.Helper()
	pool := testsupport.NewDB(t)
	cryptor := webhookCryptor(t)
	repository := NewTargetRepository(pool, cryptor, bytes.NewReader(bytes.Repeat([]byte{0x77}, 64)), nil, true)
	target, err := repository.Create(context.Background(), "receiver", endpoint, "generic")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	var eventID uuid.UUID
	err = pool.QueryRow(context.Background(), `INSERT INTO outbox_events (event_type,event_version,aggregate_type,aggregate_id,request_id,payload)
		VALUES ('tenancit.tenant.created',1,'tenant','tenant-1','request-1','{"schema_version":1}') RETURNING id`).Scan(&eventID)
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	var deliveryID uuid.UUID
	err = pool.QueryRow(context.Background(), `INSERT INTO webhook_deliveries (event_id,target_id) VALUES ($1,$2) RETURNING id`, eventID, target.ID).Scan(&deliveryID)
	if err != nil {
		t.Fatalf("delivery: %v", err)
	}
	return NewDeliveryStore(pool), cryptor, target, deliveryID
}

func TestWorkerDeliversSignedEventExactlyOnce(t *testing.T) {
	var calls atomic.Int32
	var secret string
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls.Add(1)
		if !VerifySignature([]byte(secret), r.Header.Get("Tenancit-Webhook-Timestamp"), body, r.Header.Get("Tenancit-Webhook-Signature")) {
			t.Error("invalid signature")
		}
		if r.Header.Get("Tenancit-Webhook-Id") == "" {
			t.Error("missing event id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()
	store, cryptor, target, deliveryID := seedDelivery(t, receiver.URL)
	secret = target.SigningSecret
	now := time.Now().UTC().Add(time.Minute)
	worker := NewWorker(store, cryptor)
	worker.Now = func() time.Time { return now }
	worker.Jitter = func() float64 { return 0.5 }
	worker.run(context.Background())
	worker.run(context.Background())
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
	var status string
	if err := store.pool.QueryRow(context.Background(), `SELECT status FROM webhook_deliveries WHERE id=$1`, deliveryID).Scan(&status); err != nil || status != "delivered" {
		t.Fatalf("status=%s err=%v", status, err)
	}
}

func TestWorkerMovesPermanentFailureToDLQAndReplayRecovers(t *testing.T) {
	var recoverReceiver atomic.Bool
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if recoverReceiver.Load() {
			w.WriteHeader(204)
		} else {
			w.WriteHeader(400)
		}
	}))
	defer receiver.Close()
	store, cryptor, _, deliveryID := seedDelivery(t, receiver.URL)
	now := time.Now().UTC().Add(time.Minute)
	worker := NewWorker(store, cryptor)
	worker.Now = func() time.Time { return now }
	worker.Jitter = func() float64 { return 0.5 }
	worker.run(context.Background())
	var dead int
	if err := store.pool.QueryRow(context.Background(), `SELECT count(*) FROM webhook_dead_letters WHERE delivery_id=$1`, deliveryID).Scan(&dead); err != nil || dead != 1 {
		t.Fatalf("dead=%d err=%v", dead, err)
	}
	recoverReceiver.Store(true)
	if err := store.Replay(context.Background(), deliveryID, now); err != nil {
		t.Fatalf("replay: %v", err)
	}
	worker.run(context.Background())
	var status string
	_ = store.pool.QueryRow(context.Background(), `SELECT status FROM webhook_deliveries WHERE id=$1`, deliveryID).Scan(&status)
	if status != "delivered" {
		t.Fatalf("status=%s", status)
	}
}
