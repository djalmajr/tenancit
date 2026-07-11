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
	GetResourceHeader(ctx context.Context, id uuid.UUID) (db.GetResourceHeaderRow, error)
	GetTenantByHostname(ctx context.Context, hostname string) (db.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (db.Tenant, error)
	ListResourceFieldValuesByResourceIDs(ctx context.Context, resourceIds []uuid.UUID) ([]db.ListResourceFieldValuesByResourceIDsRow, error)
	ListResourceHeadersByTenant(ctx context.Context, arg db.ListResourceHeadersByTenantParams) ([]db.ListResourceHeadersByTenantRow, error)
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

// ErrTenantUnavailable intentionally merges suspended and missing consumer
// identities at the HTTP boundary. Admin queries continue to expose inactive
// tenants so operators can reactivate them.
var ErrTenantUnavailable = errors.New("tenant unavailable for consumer resolution")

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
	hostname = CanonicalHostname(hostname)
	if hostname == "" {
		return db.Tenant{}, pgx.ErrNoRows
	}
	tenant, err := r.q.GetTenantByHostname(ctx, hostname)
	if err != nil {
		return db.Tenant{}, err
	}
	if tenant.Status != "active" {
		return db.Tenant{}, ErrTenantUnavailable
	}
	return tenant, nil
}

// TenantBySlug looks up a tenant by its canonical slug (the x-tenant-id identity).
func (r *Resolver) TenantBySlug(ctx context.Context, slug string) (db.Tenant, error) {
	tenant, err := r.q.GetTenantBySlug(ctx, slug)
	if err != nil {
		return db.Tenant{}, err
	}
	if tenant.Status != "active" {
		return db.Tenant{}, ErrTenantUnavailable
	}
	return tenant, nil
}

// Version returns a strong ETag for the tenant's resolved config plus the active
// resources used to compute it. It does NOT decrypt anything, so callers can
// answer conditional GETs (If-None-Match -> 304) cheaply, and pass the returned
// resources to ResolveTenant to avoid a second query on a cache miss.
func (r *Resolver) Version(ctx context.Context, tenant db.Tenant) (string, []ResourceHeader, error) {
	headers, err := LoadResourceHeaders(ctx, r.q, tenant.ID, false)
	if err != nil {
		return "", nil, err
	}
	return computeETag(tenant, headers), headers, nil
}

// ResolveTenant decrypts and assembles the consumer payload for already-loaded
// active resources.
func (r *Resolver) ResolveTenant(ctx context.Context, tenant db.Tenant, headers []ResourceHeader) (ResolvedTenant, error) {
	built, err := BuildResourceFieldsBatch(ctx, r.q, r.c, headers, true)
	if err != nil {
		return ResolvedTenant{}, err
	}
	out := ResolvedTenant{TenantSlug: tenant.Slug}
	for _, resource := range built {
		rr := ResolvedResource{DefinitionKey: resource.Header.DefinitionKey, Values: map[string]string{}}
		for _, field := range resource.Fields {
			rr.Values[field.Key] = field.Value
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
	headers, err := LoadResourceHeaders(ctx, r.q, tenant.ID, false)
	if err != nil {
		return ResolvedTenant{}, err
	}
	return r.ResolveTenant(ctx, tenant, headers)
}

// IdentityETag is a strong ETag for the hostname -> tenant identity mapping.
// The edge injector only needs the tenant slug (never secrets), so it resolves
// via /v1/identify and revalidates with this tag, which changes only when the
// tenant itself changes (slug/update) — not when a resource value changes.
func IdentityETag(t db.Tenant) string {
	h := sha256.New()
	fmt.Fprintf(h, "id:%s:%s:%d\n", t.ID, t.Slug, t.UpdatedAt.UnixNano())
	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`
}

// computeETag derives a stable strong ETag from the tenant and its active
// resources' identities + timestamps + statuses. Any add/remove/update/status
// change of a resource, or a tenant update, changes the tag. Order-independent
// (resources are sorted by id), so query ordering does not affect the result.
func computeETag(t db.Tenant, headers []ResourceHeader) string {
	sorted := make([]ResourceHeader, len(headers))
	copy(sorted, headers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Resource.ID.String() < sorted[j].Resource.ID.String()
	})
	h := sha256.New()
	fmt.Fprintf(h, "t:%s:%s:%d:%d\n", t.ID, t.Slug, t.UpdatedAt.UnixNano(), len(sorted))
	for _, header := range sorted {
		res := header.Resource
		fmt.Fprintf(h, "r:%s:%s:%d:d:%s:%s:%d\n",
			res.ID, res.Status, res.UpdatedAt.UnixNano(), res.ResourceDefinitionID,
			header.DefinitionKey, header.DefinitionUpdatedAt.UnixNano())
	}
	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`
}

// ByHostnameAndDefinition returns a single resource by definition key.
func (r *Resolver) ByHostnameAndDefinition(ctx context.Context, hostname, defKey string) (ResolvedResource, bool, error) {
	tenant, err := r.TenantByHostname(ctx, hostname)
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
	headerRow, err := r.q.GetResourceHeader(ctx, res.ID)
	if err != nil {
		return ResolvedResource{}, false, err
	}
	built, err := BuildResourceFieldsBatch(ctx, r.q, r.c, []ResourceHeader{resourceHeaderFromGetRow(headerRow)}, true)
	if err != nil {
		return ResolvedResource{}, false, err
	}
	rr := ResolvedResource{DefinitionKey: headerRow.DefinitionKey, Values: map[string]string{}}
	for _, field := range built[0].Fields {
		rr.Values[field.Key] = field.Value
	}
	return rr, true, nil
}
