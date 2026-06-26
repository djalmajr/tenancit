package store_test

import (
	"context"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
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
