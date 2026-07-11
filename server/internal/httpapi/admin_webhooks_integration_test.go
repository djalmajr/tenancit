package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/webhook"
)

func TestE2E_WebhookTargetEncryptsEndpointAndReturnsSecretOnce(t *testing.T) {
	server, _ := newTestServer(t)
	server.SetWebhookTargets(webhook.NewTargetRepository(server.DB, server.Cryptor, bytes.NewReader(bytes.Repeat([]byte{0x55}, 64)), nil, true))
	handler := server.Routes(nil)

	created := do(t, handler, http.MethodPost, "/v1/admin/webhook-targets", map[string]string{
		"name": "local receiver", "url": "http://127.0.0.1:19090/private-hook-path", "format": "generic",
	})
	if created.Code != http.StatusCreated || created.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("create status=%d cache=%q body=%s", created.Code, created.Header().Get("Cache-Control"), created.Body)
	}
	var oneShot webhook.CreatedTarget
	mustJSON(t, created, &oneShot)
	if oneShot.SigningSecret == "" || oneShot.Endpoint != "http://127.0.0.1:19090" {
		t.Fatalf("created=%+v", oneShot)
	}

	listed := do(t, handler, http.MethodGet, "/v1/admin/webhook-targets", nil)
	var targets []webhook.Target
	mustJSON(t, listed, &targets)
	if listed.Code != http.StatusOK || len(targets) != 1 || targets[0].Endpoint != "http://127.0.0.1:19090" {
		t.Fatalf("list status=%d targets=%+v", listed.Code, targets)
	}
	if bytes.Contains(listed.Body.Bytes(), []byte("private-hook-path")) || bytes.Contains(listed.Body.Bytes(), []byte(oneShot.SigningSecret)) {
		t.Fatalf("list leaked endpoint path or signing secret: %s", listed.Body)
	}

	var containsPlaintext bool
	if err := server.DB.QueryRow(context.Background(), `SELECT position(convert_to('private-hook-path', 'UTF8') in url_cipher) > 0 FROM webhook_targets LIMIT 1`).Scan(&containsPlaintext); err != nil {
		t.Fatalf("inspect encrypted URL: %v", err)
	}
	if containsPlaintext {
		t.Fatal("database stored webhook URL as plaintext")
	}
}

func TestE2E_WebhookOverviewAndReplayAreConsistentAndAudited(t *testing.T) {
	server, _ := newTestServer(t)
	server.SetWebhookTargets(webhook.NewTargetRepository(server.DB, server.Cryptor, bytes.NewReader(bytes.Repeat([]byte{0x44}, 64)), nil, true))
	handler := server.Routes(nil)

	created := do(t, handler, http.MethodPost, "/v1/admin/webhook-targets", map[string]string{
		"name": "replay receiver", "url": "http://127.0.0.1:19091/hook", "format": "generic",
	})
	var target webhook.CreatedTarget
	mustJSON(t, created, &target)
	var deliveryID string
	if err := server.DB.QueryRow(context.Background(), `WITH event AS (
		INSERT INTO outbox_events (event_type,event_version,aggregate_type,aggregate_id,request_id,payload)
		VALUES ('tenancit.tenant.created',1,'tenant','tenant-replay','request-replay','{}') RETURNING id
	) INSERT INTO webhook_deliveries (event_id,target_id,status,attempt_count,last_error_code)
	SELECT id,$1,'dead_letter',8,'attempts_exhausted' FROM event RETURNING id`, target.ID).Scan(&deliveryID); err != nil {
		t.Fatalf("seed dead-letter delivery: %v", err)
	}
	if _, err := server.DB.Exec(context.Background(), `INSERT INTO webhook_dead_letters (delivery_id,reason_code) VALUES ($1,'attempts_exhausted')`, deliveryID); err != nil {
		t.Fatalf("seed dead letter: %v", err)
	}

	overviewResponse := do(t, handler, http.MethodGet, "/v1/admin/webhook-overview", nil)
	var before webhookOverview
	mustJSON(t, overviewResponse, &before)
	if overviewResponse.Code != http.StatusOK || before.Targets != 1 || before.DeadLetter != 1 {
		t.Fatalf("overview status=%d body=%+v", overviewResponse.Code, before)
	}

	replayed := do(t, handler, http.MethodPost, "/v1/admin/webhook-deliveries/"+deliveryID+"/replay", nil)
	if replayed.Code != http.StatusNoContent {
		t.Fatalf("replay status=%d body=%s", replayed.Code, replayed.Body)
	}
	var status string
	var attempts int
	if err := server.DB.QueryRow(context.Background(), `SELECT status,attempt_count FROM webhook_deliveries WHERE id=$1`, deliveryID).Scan(&status, &attempts); err != nil {
		t.Fatalf("read replayed delivery: %v", err)
	}
	if status != "pending" || attempts != 0 {
		t.Fatalf("status=%s attempts=%d", status, attempts)
	}
	var auditCount int
	if err := server.DB.QueryRow(context.Background(), `SELECT count(*) FROM admin_audit_events WHERE action='webhook_delivery.replayed' AND target_id=$1`, deliveryID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}

	conflict := do(t, handler, http.MethodPost, "/v1/admin/webhook-deliveries/"+deliveryID+"/replay", nil)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("second replay status=%d body=%s", conflict.Code, conflict.Body)
	}
}
