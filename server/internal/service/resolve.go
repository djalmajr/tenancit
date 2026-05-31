package service

import (
	"context"
	"fmt"

	"github.com/centralit/resource-tenant/server/internal/crypto"
	"github.com/centralit/resource-tenant/server/internal/store/db"
	"github.com/google/uuid"
)

// ResolveQuerier is the read surface needed to resolve a tenant's config.
type ResolveQuerier interface {
	GetTenantByHostname(ctx context.Context, hostname string) (db.Tenant, error)
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

// ByHostname returns all active resources for the tenant owning the hostname.
func (r *Resolver) ByHostname(ctx context.Context, hostname string) (ResolvedTenant, error) {
	tenant, err := r.q.GetTenantByHostname(ctx, hostname)
	if err != nil {
		return ResolvedTenant{}, fmt.Errorf("resolve hostname %q: %w", hostname, err)
	}
	resources, err := r.q.ListActiveResourcesByTenant(ctx, tenant.ID)
	if err != nil {
		return ResolvedTenant{}, err
	}
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

// ByHostnameAndDefinition returns a single resource by definition key.
func (r *Resolver) ByHostnameAndDefinition(ctx context.Context, hostname, defKey string) (ResolvedResource, bool, error) {
	full, err := r.ByHostname(ctx, hostname)
	if err != nil {
		return ResolvedResource{}, false, err
	}
	for _, rr := range full.Resources {
		if rr.DefinitionKey == defKey {
			return rr, true, nil
		}
	}
	return ResolvedResource{}, false, nil
}

func (r *Resolver) resolveResource(ctx context.Context, res db.TenantResource) (ResolvedResource, error) {
	def, err := r.q.GetDefinition(ctx, res.ResourceDefinitionID)
	if err != nil {
		return ResolvedResource{}, err
	}
	fields, err := r.q.ListFields(ctx, def.ID)
	if err != nil {
		return ResolvedResource{}, err
	}
	fieldByID := make(map[uuid.UUID]db.ResourceField, len(fields))
	for _, f := range fields {
		fieldByID[f.ID] = f
	}
	values, err := r.q.ListResourceValues(ctx, res.ID)
	if err != nil {
		return ResolvedResource{}, err
	}
	rr := ResolvedResource{DefinitionKey: def.Key, Values: map[string]string{}}
	for _, v := range values {
		f, ok := fieldByID[v.ResourceFieldID]
		if !ok {
			continue
		}
		cleartext, err := decodeValue(r.c, v)
		if err != nil {
			return ResolvedResource{}, err
		}
		rr.Values[f.Key] = cleartext
	}
	return rr, nil
}
