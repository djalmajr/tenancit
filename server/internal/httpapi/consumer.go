package httpapi

import (
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
)

// resolveMaxAge is the Cache-Control max-age (seconds) advertised on /v1/resolve.
// Short by design: clients revalidate cheaply via the ETag (304) rather than
// holding stale config; tenant/resource edits flip the ETag immediately.
const resolveMaxAge = "private, max-age=30"

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
	case tenantID != "":
		tenant, err = s.Resolver.TenantBySlug(ctx, tenantID)
	case hostname != "":
		tenant, err = s.Resolver.TenantByHostname(ctx, hostname)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname or tenantId required"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}

	etag, resources, err := s.Resolver.Version(ctx, tenant)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", resolveMaxAge)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	res, err := s.Resolver.ResolveTenant(ctx, tenant, resources)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleResolveOne(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	defKey := chi.URLParam(r, "definitionKey")
	res, found, err := s.Resolver.ByHostnameAndDefinition(r.Context(), hostname, defKey)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
