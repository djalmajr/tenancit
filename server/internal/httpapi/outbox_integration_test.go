package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestE2E_DomainMutationPublishesReferenceOnlyOutboxAndDelivery(t *testing.T) {
	server, handler := newTestServer(t)
	ctx := context.Background()
	if _, err := server.DB.Exec(ctx, `
		INSERT INTO webhook_targets (
			name, url_cipher, url_nonce, url_key_version,
			signing_secret_cipher, signing_secret_nonce, signing_secret_key_version
		) VALUES ('receiver', '\x01', '\x02', 1, '\x03', '\x04', 1)
	`); err != nil {
		t.Fatalf("insert target: %v", err)
	}

	created := do(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{
		"slug": "outbox-tenant", "name": "Sensitive customer name",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body)
	}
	tenantID := idOf(t, created)

	var eventType, aggregateType, aggregateID, payload string
	if err := server.DB.QueryRow(ctx, `
		SELECT event_type, aggregate_type, aggregate_id, payload::text
		FROM outbox_events
		WHERE aggregate_type = 'tenant' AND aggregate_id = $1
	`, tenantID).Scan(&eventType, &aggregateType, &aggregateID, &payload); err != nil {
		t.Fatalf("read outbox event: %v", err)
	}
	if eventType != "tenancit.tenant.created" || aggregateType != "tenant" || aggregateID != tenantID {
		t.Fatalf("event=%s aggregate=%s/%s", eventType, aggregateType, aggregateID)
	}
	for _, forbidden := range []string{"Sensitive customer name", "secret", "token", "password"} {
		if strings.Contains(strings.ToLower(payload), strings.ToLower(forbidden)) {
			t.Fatalf("payload leaked %q: %s", forbidden, payload)
		}
	}
	var deliveries int
	if err := server.DB.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries`).Scan(&deliveries); err != nil || deliveries != 1 {
		t.Fatalf("deliveries=%d err=%v", deliveries, err)
	}
}

func TestE2E_OutboxFailureRollsBackDomainAndAudit(t *testing.T) {
	server, handler := newTestServer(t)
	ctx := context.Background()
	if _, err := server.DB.Exec(ctx, `
		CREATE FUNCTION fail_outbox_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'outbox unavailable'; END $$;
		CREATE TRIGGER fail_outbox_insert BEFORE INSERT ON outbox_events
		FOR EACH ROW EXECUTE FUNCTION fail_outbox_insert()
	`); err != nil {
		t.Fatalf("install outbox failure: %v", err)
	}

	created := do(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{
		"slug": "must-rollback", "name": "Must rollback",
	})
	if created.Code != http.StatusInternalServerError {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body)
	}
	var tenants, audits int
	if err := server.DB.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE slug = 'must-rollback'`).Scan(&tenants); err != nil {
		t.Fatalf("count tenant: %v", err)
	}
	if err := server.DB.QueryRow(ctx, `SELECT count(*) FROM admin_audit_events WHERE target_type = 'tenant'`).Scan(&audits); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if tenants != 0 || audits != 0 {
		t.Fatalf("transaction leaked tenants=%d audits=%d", tenants, audits)
	}
}
