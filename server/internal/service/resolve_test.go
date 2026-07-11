package service

import (
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
)

func TestComputeETagIsOrderIndependentAndTracksResourceState(t *testing.T) {
	base := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	tenant := db.Tenant{ID: uuid.New(), Slug: "acme", UpdatedAt: base}
	a := db.TenantResource{
		ID: uuid.New(), ResourceDefinitionID: uuid.New(), Status: "active", UpdatedAt: base,
	}
	b := db.TenantResource{
		ID: uuid.New(), ResourceDefinitionID: uuid.New(), Status: "active", UpdatedAt: base.Add(time.Second),
	}
	ha := ResourceHeader{Resource: a, DefinitionKey: "postgres", DefinitionUpdatedAt: base}
	hb := ResourceHeader{Resource: b, DefinitionKey: "redis", DefinitionUpdatedAt: base}

	first := computeETag(tenant, []ResourceHeader{ha, hb})
	if got := computeETag(tenant, []ResourceHeader{hb, ha}); got != first {
		t.Fatalf("ETag depends on resource order: %q != %q", got, first)
	}
	hb.Resource.Status = "inactive"
	if got := computeETag(tenant, []ResourceHeader{ha, hb}); got == first {
		t.Fatal("ETag did not change with resource status")
	}
	hb.Resource.Status = "active"
	hb.Resource.UpdatedAt = hb.Resource.UpdatedAt.Add(time.Second)
	if got := computeETag(tenant, []ResourceHeader{ha, hb}); got == first {
		t.Fatal("ETag did not change with resource updated_at")
	}
	hb.Resource.UpdatedAt = b.UpdatedAt
	hb.DefinitionUpdatedAt = base.Add(time.Second)
	if got := computeETag(tenant, []ResourceHeader{ha, hb}); got == first {
		t.Fatal("ETag did not change with definition updated_at")
	}
}

func TestIdentityETagTracksTenantIdentity(t *testing.T) {
	base := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	tenant := db.Tenant{ID: uuid.New(), Slug: "acme", UpdatedAt: base}
	etag := IdentityETag(tenant)
	tenant.Slug = "acme-renamed"
	if IdentityETag(tenant) == etag {
		t.Fatal("identity ETag did not change with slug")
	}
	tenant.Slug = "acme"
	tenant.UpdatedAt = base.Add(time.Second)
	if IdentityETag(tenant) == etag {
		t.Fatal("identity ETag did not change with updated_at")
	}
}
