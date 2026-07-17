package httpapi

import (
	"errors"
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
)

// Resolve payloads contain cleartext credentials. Validators may be retained
// by application code, but the HTTP response body must never be stored.
const secretResponseCacheControl = "private, no-store"

// identifyCacheControl lets a private cache retain the identity only for cheap
// conditional revalidation. Hostnames can be reassigned between tenants, so a
// freshness window could serve the previous tenant without contacting us.
const identifyCacheControl = "private, no-cache"

// identifyResponse is the edge-facing payload: tenant identity only, no secrets.
type identifyResponse struct {
	TenantSlug string `json:"tenantSlug"`
}

// handleIdentify maps a hostname to its tenant slug for the edge injector, which
// injects x-tenant-id and must never touch credentials. Emits an ETag and
// answers If-None-Match with 304.
func (s *Server) handleIdentify(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}
	tenant, err := s.Resolver.TenantByHostname(r.Context(), hostname)
	if err != nil {
		if tenantUnavailable(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
			return
		}
		writeInternalError(w, r, "identify tenant", err)
		return
	}
	etag := service.IdentityETag(tenant)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", identifyCacheControl)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, identifyResponse{TenantSlug: tenant.Slug})
}

// handleResolve resolves a tenant's active resources either by ?hostname= (edge
// injector) or by ?tenantId=<slug> (the app resolving its own DB by identity).
// It emits a strong ETag and answers conditional GETs with 304 without
// decrypting, so repeated identical lookups are cheap.
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hostname := r.URL.Query().Get("hostname")
	tenantID := r.URL.Query().Get("tenantId")

	var (
		tenant db.Tenant
		err    error
	)
	switch {
	case tenantID != "" && hostname != "":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provide only one of hostname or tenantId"})
		return
	case tenantID != "":
		tenant, err = s.Resolver.TenantBySlug(ctx, tenantID)
	case hostname != "":
		tenant, err = s.Resolver.TenantByHostname(ctx, hostname)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname or tenantId required"})
		return
	}
	if err != nil {
		if tenantUnavailable(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
			return
		}
		writeInternalError(w, r, "load tenant for resolve", err)
		return
	}

	etag, resources, err := s.Resolver.Version(ctx, tenant)
	if err != nil {
		writeInternalError(w, r, "version resolved tenant", err)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", secretResponseCacheControl)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	res, err := s.Resolver.ResolveTenant(ctx, tenant, resources)
	if err != nil {
		writeInternalError(w, r, "resolve tenant resources", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleResolveOne(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	alias := chi.URLParam(r, "alias")
	res, found, err := s.Resolver.ByHostnameAndAlias(r.Context(), hostname, alias)
	if err != nil {
		if tenantUnavailable(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
			return
		}
		writeInternalError(w, r, "resolve tenant resource", err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}
	w.Header().Set("Cache-Control", secretResponseCacheControl)
	writeJSON(w, http.StatusOK, res)
}

func tenantUnavailable(err error) bool {
	return isNotFound(err) || errors.Is(err, service.ErrTenantUnavailable)
}

// tenantDirectoryEntry is the consumer-facing tenant identity: directory data
// only (slug/name/status), never resources or secrets.
type tenantDirectoryEntry struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// handleListTenants enumerates tenant identities for consumers holding the
// tenant:list scope. Control-plane bridges (e.g. platform shells) use it to
// mirror the tenant directory without admin credentials.
func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := s.Q.ListTenants(r.Context())
	if err != nil {
		writeInternalError(w, r, "list tenants for consumer", err)
		return
	}
	entries := make([]tenantDirectoryEntry, 0, len(tenants))
	for _, tenant := range tenants {
		entries = append(entries, tenantDirectoryEntry{
			Slug:   tenant.Slug,
			Name:   tenant.Name,
			Status: tenant.Status,
		})
	}
	w.Header().Set("Cache-Control", identifyCacheControl)
	writeJSON(w, http.StatusOK, entries)
}
