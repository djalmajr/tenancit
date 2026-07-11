package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Schema invariants verified against a real Postgres (testcontainers).

// RN-01: at most one ACTIVE resource per (tenant, definition).
// Mutation captured: dropping the partial unique index uq_tenant_resource_active
// lets the second active insert succeed.
func TestSchema_OneActiveResourcePerType(t *testing.T) {
	pool := testsupport.NewDB(t)
	q := db.New(pool)
	ctx := context.Background()

	tenant, err := q.CreateTenant(ctx, db.CreateTenantParams{Slug: "acme", Name: "Acme"})
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	def, err := q.CreateDefinition(ctx, db.CreateDefinitionParams{Key: "pg", Name: "PG"})
	if err != nil {
		t.Fatalf("def: %v", err)
	}
	if _, err := q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: tenant.ID, ResourceDefinitionID: def.ID,
	}); err != nil {
		t.Fatalf("first active: %v", err)
	}
	if _, err := q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: tenant.ID, ResourceDefinitionID: def.ID,
	}); err == nil {
		t.Fatal("expected unique violation for 2nd active resource (RN-01)")
	}
}

// tenant_domains.hostname is globally unique (1 hostname -> 1 tenant).
// Mutation captured: dropping UNIQUE on hostname allows the duplicate insert.
func TestSchema_HostnameUnique(t *testing.T) {
	pool := testsupport.NewDB(t)
	q := db.New(pool)
	ctx := context.Background()

	a, _ := q.CreateTenant(ctx, db.CreateTenantParams{Slug: "a", Name: "A"})
	b, _ := q.CreateTenant(ctx, db.CreateTenantParams{Slug: "b", Name: "B"})
	if _, err := q.AddTenantDomain(ctx, db.AddTenantDomainParams{TenantID: a.ID, Hostname: "x.example.com"}); err != nil {
		t.Fatalf("first domain: %v", err)
	}
	if _, err := q.AddTenantDomain(ctx, db.AddTenantDomainParams{TenantID: b.ID, Hostname: "x.example.com"}); err == nil {
		t.Fatal("expected unique violation for duplicate hostname")
	}
}

// resource_fields (definition_id, key) is unique.
// Mutation captured: dropping the composite UNIQUE allows duplicate field keys.
func TestSchema_FieldKeyUniquePerDefinition(t *testing.T) {
	pool := testsupport.NewDB(t)
	q := db.New(pool)
	ctx := context.Background()

	def, _ := q.CreateDefinition(ctx, db.CreateDefinitionParams{Key: "pg", Name: "PG"})
	if _, err := q.AddField(ctx, db.AddFieldParams{ResourceDefinitionID: def.ID, Key: "host", DataType: "string"}); err != nil {
		t.Fatalf("first field: %v", err)
	}
	if _, err := q.AddField(ctx, db.AddFieldParams{ResourceDefinitionID: def.ID, Key: "host", DataType: "string"}); err == nil {
		t.Fatal("expected unique violation for duplicate field key")
	}
}

func TestSchema_EnforcesCanonicalHostnameAndTenantStatus(t *testing.T) {
	pool := testsupport.NewDB(t)
	q := db.New(pool)
	ctx := context.Background()

	tenant, err := q.CreateTenant(ctx, db.CreateTenantParams{Slug: "canonical", Name: "Canonical"})
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	if _, err := q.AddTenantDomain(ctx, db.AddTenantDomainParams{
		TenantID: tenant.ID, Hostname: "Mixed.Example.com",
	}); err == nil {
		t.Fatal("database accepted a non-canonical hostname")
	}
	if _, err := q.UpdateTenant(ctx, db.UpdateTenantParams{
		ID: tenant.ID, Name: tenant.Name, Slug: tenant.Slug, Status: "paused",
	}); err == nil {
		t.Fatal("database accepted a tenant status outside active|inactive")
	}
}

func TestSchema_RejectsLegacyUnboundedAPIClients(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO api_clients (name, key_hash)
		VALUES ('legacy', 'legacy-hash')
	`); err == nil {
		t.Fatal("database accepted an unbounded API client")
	}
}

func TestSchema_APIClientRevocationIsTerminal(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()

	var clientID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO api_clients (name, key_hash, token_preview, rpm_limit, expires_at)
		VALUES ('terminal', 'terminal-hash', 'tnc_...term', 100, now() + interval '1 day')
		RETURNING id
	`).Scan(&clientID); err != nil {
		t.Fatalf("insert governed client: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE api_clients SET status = 'revoked' WHERE id = $1`, clientID); err != nil {
		t.Fatalf("revoke client: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE api_clients SET status = 'active' WHERE id = $1`, clientID); err == nil {
		t.Fatal("database allowed a revoked API client to reactivate")
	}
}

func TestSchema_AdminAuditEventsAreAppendOnly(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()

	var occurredAt string
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO admin_audit_events (
			request_id, actor_kind, actor_subject, action, target_type,
			target_id, result, http_method, route_template, http_status
		) VALUES ('req-1', 'shared_admin_token', 'primary', 'tenant.created',
			'tenant', 'target-1', 'success', 'POST', '/v1/admin/tenants', 201)
		RETURNING occurred_at::text, id::text
	`).Scan(&occurredAt, &id); err != nil {
		t.Fatalf("insert audit event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE admin_audit_events SET result = 'error'
		WHERE occurred_at = $1::timestamptz AND id = $2::uuid
	`, occurredAt, id); err == nil {
		t.Fatal("database allowed an audit event update")
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM admin_audit_events
		WHERE occurred_at = $1::timestamptz AND id = $2::uuid
	`, occurredAt, id); err == nil {
		t.Fatal("database allowed an audit event delete")
	}
}

func TestSchema_APIClientUsageCoalescesAndNeverMovesLastUsedBackward(t *testing.T) {
	pool := testsupport.NewDB(t)
	q := db.New(pool)
	ctx := context.Background()
	var clientID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO api_clients (name, key_hash, token_preview, rpm_limit, expires_at)
		VALUES ('usage', 'usage-hash', 'tnc_...test', 100, now() + interval '1 day')
		RETURNING id
	`).Scan(&clientID); err != nil {
		t.Fatalf("insert client: %v", err)
	}
	newer := time.Date(2026, 7, 10, 12, 5, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	for _, usedAt := range []time.Time{newer, older} {
		if err := q.TouchAPIClientLastUsed(ctx, db.TouchAPIClientLastUsedParams{
			ApiClientID: clientID, UsedAt: usedAt,
		}); err != nil {
			t.Fatalf("touch last used: %v", err)
		}
	}
	var stored time.Time
	if err := pool.QueryRow(ctx, `SELECT last_used_at FROM api_clients WHERE id = $1`, clientID).Scan(&stored); err != nil {
		t.Fatalf("read last used: %v", err)
	}
	if !stored.Equal(newer) {
		t.Fatalf("last_used_at = %v, want %v", stored, newer)
	}
	day := pgtype.Date{Time: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Valid: true}
	for _, count := range []int64{2, 3} {
		if err := q.UpsertAPIClientUsageDaily(ctx, db.UpsertAPIClientUsageDailyParams{
			Day: day, ApiClientID: clientID, Operation: "resolve", StatusClass: 2, RequestCount: count,
		}); err != nil {
			t.Fatalf("upsert usage: %v", err)
		}
	}
	var requests int64
	if err := pool.QueryRow(ctx, `
		SELECT request_count FROM api_client_usage_daily
		WHERE day = '2026-07-10' AND api_client_id = $1 AND operation = 'resolve' AND status_class = 2
	`, clientID).Scan(&requests); err != nil {
		t.Fatalf("read usage: %v", err)
	}
	if requests != 5 {
		t.Fatalf("request_count = %d, want 5", requests)
	}
}
