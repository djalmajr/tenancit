package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB test")
	}
	return dsn
}

func TestMigrate_CreatesSchemaAndInvariant(t *testing.T) {
	dsn := testDSN(t)
	if err := Migrate(dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	for _, tbl := range []string{
		"tenants", "tenant_domains", "resource_definitions", "resource_fields",
		"tenant_resources", "tenant_resource_values", "api_clients",
	} {
		var exists bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, tbl,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %q missing (err=%v)", tbl, err)
		}
	}

	var tenantID, defID string
	if err := conn.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('acme','Acme') RETURNING id`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	if err := conn.QueryRow(ctx, `INSERT INTO resource_definitions (key, name) VALUES ('postgres','PG') RETURNING id`).Scan(&defID); err != nil {
		t.Fatalf("def: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tenant_resources (tenant_id, resource_definition_id, status) VALUES ($1,$2,'active')`, tenantID, defID); err != nil {
		t.Fatalf("first active: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tenant_resources (tenant_id, resource_definition_id, status) VALUES ($1,$2,'active')`, tenantID, defID); err == nil {
		t.Fatal("expected unique violation for 2nd active (RN-01)")
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tenant_resources (tenant_id, resource_definition_id, status) VALUES ($1,$2,'inactive')`, tenantID, defID); err != nil {
		t.Fatalf("inactive should be allowed: %v", err)
	}
}
