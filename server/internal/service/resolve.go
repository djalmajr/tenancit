package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResolveQuerier is the read surface needed to resolve a tenant's config.
type ResolveQuerier interface {
	GetActiveResourceByTenantAndDefinitionKey(ctx context.Context, arg db.GetActiveResourceByTenantAndDefinitionKeyParams) (db.TenantResource, error)
	GetTenantByHostname(ctx context.Context, hostname string) (db.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (db.Tenant, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (db.ResourceDefinition, error)
	ListFields(ctx context.Context, resourceDefinitionID uuid.UUID) ([]db.ResourceField, error)
	ListActiveResourcesByTenant(ctx context.Context, tenantID uuid.UUID) ([]db.TenantResource, error)
	ListResourceValues(ctx context.Context, tenantResourceID uuid.UUID) ([]db.TenantResourceValue, error)
}

// ResolvedResource is one resource (definition + decrypted values) for a tenant.
type ResolvedResource struct {
	DefinitionKey string            `json:"definitionKey"`
	Values        map[string]string `json:"values"`
}

// ResolvedTenant is the consumer-facing payload for a hostname.
type ResolvedTenant struct {
	TenantSlug string             `json:"tenantSlug"`
	Resources  []ResolvedResource `json:"resources"`
}

// Resolver turns a hostname into active resources with cleartext values
// (RN-02 exact hostname, RN-05 decrypt server-side).
type Resolver struct {
	q ResolveQuerier
	c *crypto.Cryptor
}

func NewResolver(q ResolveQuerier, c *crypto.Cryptor) *Resolver {
	return &Resolver{q: q, c: c}
}

// TenantByHostname looks up the tenant owning a hostname (RN-02 exact match).
func (r *Resolver) TenantByHostname(ctx context.Context, hostname string) (db.Tenant, error) {
	return r.q.GetTenantByHostname(ctx, hostname)
}

// TenantBySlug looks up a tenant by its canonical slug (the x-tenant-id identity).
func (r *Resolver) TenantBySlug(ctx context.Context, slug string) (db.Tenant, error) {
	return r.q.GetTenantBySlug(ctx, slug)
}

// Version returns a strong ETag for the tenant's resolved config plus the active
// resources used to compute it. It does NOT decrypt anything, so callers can
// answer conditional GETs (If-None-Match -> 304) cheaply, and pass the returned
// resources to ResolveTenant to avoid a second query on a cache miss.
func (r *Resolver) Version(ctx context.Context, tenant db.Tenant) (string, []db.TenantResource, error) {
	resources, err := r.q.ListActiveResourcesByTenant(ctx, tenant.ID)
	if err != nil {
		return "", nil, err
	}
	return computeETag(tenant, resources), resources, nil
}

// ResolveTenant decrypts and assembles the consumer payload for already-loaded
// active resources.
func (r *Resolver) ResolveTenant(ctx context.Context, tenant db.Tenant, resources []db.TenantResource) (ResolvedTenant, error) {
	out := ResolvedTenant{TenantSlug: tenant.Slug}
	for _, res := range resources {
		rr, err := r.resolveResource(ctx, res)
		if err != nil {
			return ResolvedTenant{}, err
		}
		out.Resources = append(out.Resources, rr)
	}
	return out, nil
}

// ByHostname returns all active resources for the tenant owning the hostname.
func (r *Resolver) ByHostname(ctx context.Context, hostname string) (ResolvedTenant, error) {
	tenant, err := r.TenantByHostname(ctx, hostname)
	if err != nil {
		return ResolvedTenant{}, fmt.Errorf("resolve hostname %q: %w", hostname, err)
	}
	resources, err := r.q.ListActiveResourcesByTenant(ctx, tenant.ID)
	if err != nil {
		return ResolvedTenant{}, err
	}
	return r.ResolveTenant(ctx, tenant, resources)
}

// computeETag derives a stable strong ETag from the tenant and its active
// resources' identities + timestamps + statuses. Any add/remove/update/status
// change of a resource, or a tenant update, changes the tag. Order-independent
// (resources are sorted by id), so query ordering does not affect the result.
func computeETag(t db.Tenant, resources []db.TenantResource) string {
	sorted := make([]db.TenantResource, len(resources))
	copy(sorted, resources)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID.String() < sorted[j].ID.String() })
	h := sha256.New()
	fmt.Fprintf(h, "t:%s:%d:%d\n", t.ID, t.UpdatedAt.UnixNano(), len(sorted))
	for _, res := range sorted {
		fmt.Fprintf(h, "r:%s:%s:%d\n", res.ID, res.Status, res.UpdatedAt.UnixNano())
	}
	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`
}

// ByHostnameAndDefinition returns a single resource by definition key.
func (r *Resolver) ByHostnameAndDefinition(ctx context.Context, hostname, defKey string) (ResolvedResource, bool, error) {
	tenant, err := r.q.GetTenantByHostname(ctx, hostname)
	if err != nil {
		return ResolvedResource{}, false, err
	}
	res, err := r.q.GetActiveResourceByTenantAndDefinitionKey(ctx, db.GetActiveResourceByTenantAndDefinitionKeyParams{
		Key: defKey, TenantID: tenant.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ResolvedResource{}, false, nil
	}
	if err != nil {
		return ResolvedResource{}, false, err
	}
	rr, err := r.resolveResource(ctx, res)
	return rr, true, err
}

func (r *Resolver) resolveResource(ctx context.Context, res db.TenantResource) (ResolvedResource, error) {
	built, err := BuildResourceFields(ctx, BuildResourceFieldsDeps{
		Cryptor: r.c, Queries: r.q,
	}, BuildResourceFieldsInput{Resource: res, Reveal: true})
	if err != nil {
		return ResolvedResource{}, err
	}
	rr := ResolvedResource{DefinitionKey: built.Definition.Key, Values: map[string]string{}}
	for _, f := range built.Fields {
		rr.Values[f.Key] = f.Value
	}
	return rr, nil
}
