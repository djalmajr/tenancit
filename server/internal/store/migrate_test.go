package store_test

import (
	"context"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/migration"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func TestMigrate_CreatesSchemaAndInvariant(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, pool.Config().ConnString())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	for _, tbl := range []string{
		"tenants", "tenant_domains", "resource_definitions", "resource_fields",
		"tenant_resources", "tenant_resource_values", "api_clients",
		"api_client_scopes", "api_client_usage_daily", "admin_audit_events",
	} {
		var exists bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, tbl,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %q missing (err=%v)", tbl, err)
		}
	}

	for _, column := range []string{
		"token_preview", "rpm_limit", "expires_at", "last_used_at", "revoked_at", "updated_at",
	} {
		var exists bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'api_clients' AND column_name = $1
			)`, column).Scan(&exists); err != nil || !exists {
			t.Fatalf("api_clients column %q missing (err=%v)", column, err)
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

func TestRuntimeGrantsSeparateRuntimeJobsAndBackupCapabilities(t *testing.T) {
	pool := testsupport.NewDB(t)
	ctx := context.Background()
	if err := migration.ApplyRuntimeGrants(pool.Config().ConnString()); err != nil {
		t.Fatalf("apply grants: %v", err)
	}
	conn, err := pgx.Connect(ctx, pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `SET ROLE tenancit_runtime`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT count(*) FROM tenants`); err != nil {
		t.Fatalf("runtime cannot read domain tables: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE runtime_must_not_create_schema (id integer)`); err == nil {
		t.Fatal("runtime role created schema")
	}
	if _, err := conn.Exec(ctx, `UPDATE admin_audit_events SET result='error'`); err == nil {
		t.Fatal("runtime role mutated append-only audit")
	}
	if _, err := conn.Exec(ctx, `RESET ROLE`); err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Exec(ctx, `SET ROLE tenancit_backup`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT count(*) FROM tenants`); err != nil {
		t.Fatalf("backup cannot read tables: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tenants (slug,name) VALUES ('backup-write','forbidden')`); err == nil {
		t.Fatal("backup role wrote domain data")
	}
	if _, err := conn.Exec(ctx, `RESET ROLE`); err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Exec(ctx, `SET ROLE tenancit_jobs`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT count(*) FROM webhook_deliveries`); err != nil {
		t.Fatalf("jobs cannot read delivery queue: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO tenants (slug,name) VALUES ('jobs-write','forbidden')`); err == nil {
		t.Fatal("jobs role wrote domain data")
	}
}
