package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func parseParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}

// PATCH /v1/admin/tenants/{id}/resources/{resourceId} body {name, alias}
func (s *Server) updateResourceIdentity(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	resourceID, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	var in struct {
		Name  string `json:"name"`
		Alias string `json:"alias"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	name, alias, err := service.NormalizeResourceIdentity(in.Name, in.Alias)
	if err != nil || name == "" || alias == "" {
		switch {
		case errors.Is(err, service.ErrInvalidResourceAlias), alias == "":
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resource alias"})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resource name"})
		}
		return
	}

	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource identity update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetTenantResource(r.Context(), db.GetTenantResourceParams{ID: resourceID, TenantID: tenantID})
	if err != nil {
		writeNotFound(w)
		return
	}
	resource, err := q.UpdateTenantResourceIdentity(r.Context(), db.UpdateTenantResourceIdentityParams{
		ID: resourceID, TenantID: tenantID, DisplayName: name, Alias: alias,
	})
	if err != nil {
		switch {
		case isNotFound(err):
			writeNotFound(w)
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "resource alias already exists"})
		default:
			writeInternalError(w, r, "update resource identity", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "resource.updated", "resource", resourceID.String(),
		"/v1/admin/tenants/{id}/resources/{resourceId}", http.StatusOK,
		map[string]any{
			"tenant_id": tenantID.String(),
			"before":    map[string]string{"name": previous.DisplayName, "alias": previous.Alias},
			"after":     map[string]string{"name": resource.DisplayName, "alias": resource.Alias},
		}); err != nil {
		writeInternalError(w, r, "audit resource identity update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource identity update", err)
		return
	}
	writeJSON(w, http.StatusOK, resource)
}

// PUT /v1/admin/tenants/{id}/resources/{resourceId}/fields/{fieldKey} body {value}
func (s *Server) updateResourceField(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	resourceID, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	fieldKey := chi.URLParam(r, "fieldKey")
	if fieldKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field key required"})
		return
	}
	var in struct {
		Value string `json:"value"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource field update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	field, err := service.UpdateResourceFieldInTx(r.Context(), q, s.Cryptor, service.UpdateResourceFieldInput{
		FieldKey: fieldKey, ResourceID: resourceID, TenantID: tenantID, Value: in.Value,
	})
	if err != nil {
		var missing service.MissingRequiredFieldError
		var invalid service.InvalidFieldValueError
		switch {
		case errors.Is(err, service.ErrUnknownResource):
			writeNotFound(w)
		case errors.Is(err, service.ErrUnknownResourceField):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown resource field"})
		case errors.As(err, &missing):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": missing.Error()})
		case errors.As(err, &invalid):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": invalid.Error()})
		default:
			writeInternalError(w, r, "update resource field", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "resource.field_updated", "resource", resourceID.String(),
		"/v1/admin/tenants/{id}/resources/{resourceId}/fields/{fieldKey}", http.StatusNoContent,
		map[string]any{"tenant_id": tenantID.String(), "field_key": field.Key, "is_secret": field.IsSecret}); err != nil {
		writeInternalError(w, r, "audit resource field update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource field update", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /v1/admin/tenants/{id}/resources/{resourceId}/fields/{fieldKey}
// removes a local override so the linked resource inherits the source again.
func (s *Server) clearResourceFieldOverride(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	resourceID, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	fieldKey := chi.URLParam(r, "fieldKey")
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource override clear", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	field, err := service.ClearResourceFieldOverrideInTx(r.Context(), q, service.UpdateResourceFieldInput{
		FieldKey: fieldKey, ResourceID: resourceID, TenantID: tenantID,
	})
	if err != nil {
		var missing service.MissingRequiredFieldError
		switch {
		case errors.Is(err, service.ErrUnknownResource):
			writeNotFound(w)
		case errors.Is(err, service.ErrUnknownResourceField), errors.Is(err, service.ErrInvalidResourceSource):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field cannot inherit from source"})
		case errors.As(err, &missing):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": missing.Error()})
		default:
			writeInternalError(w, r, "clear resource field override", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "resource.field_override_cleared", "resource", resourceID.String(),
		"/v1/admin/tenants/{id}/resources/{resourceId}/fields/{fieldKey}", http.StatusNoContent,
		map[string]any{"tenant_id": tenantID.String(), "field_key": field.Key, "is_secret": field.IsSecret}); err != nil {
		writeInternalError(w, r, "audit resource override clear", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource override clear", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/tenants/{id} — update name/slug/status.
func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct{ Name, Slug, Status string }
	if !decodeJSON(w, r, &in) {
		return
	}
	slug, slugOK := service.NormalizeTenantSlug(in.Slug)
	if !slugOK {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant slug"})
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
		return
	}
	in.Slug = slug
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Status != "active" && in.Status != "inactive" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetTenant(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	t, err := q.UpdateTenant(r.Context(), db.UpdateTenantParams{
		ID: id, Name: in.Name, Slug: in.Slug, Status: in.Status,
	})
	if err != nil {
		switch {
		case isNotFound(err):
			writeNotFound(w)
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "a tenant with this slug already exists"})
		default:
			writeInternalError(w, r, "update tenant", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "tenant.updated", "tenant", id.String(),
		"/v1/admin/tenants/{id}", http.StatusOK,
		map[string]any{"before": map[string]string{"name": previous.Name, "slug": previous.Slug, "status": previous.Status}, "after": map[string]string{"name": t.Name, "slug": t.Slug, "status": t.Status}}); err != nil {
		writeInternalError(w, r, "audit tenant update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant update", err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// DELETE /v1/admin/tenants/{id} — remove a tenant and (via FK cascade) its
// domains, resources and resource values. Returns 404 if it does not exist.
func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	tenant, err := q.GetTenant(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	children, err := q.CountTenantChildren(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, "count tenant children", err)
		return
	}
	n, err := q.DeleteTenant(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, "delete tenant", err)
		return
	}
	if n == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "tenant.deleted", "tenant", id.String(),
		"/v1/admin/tenants/{id}", http.StatusNoContent,
		map[string]any{"slug": tenant.Slug, "domain_count": children.Domains, "resource_count": children.Resources}); err != nil {
		writeInternalError(w, r, "audit tenant delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/tenants/{id}/domains/{domainId} body {hostname}
func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	domainID, err := parseParam(r, "domainId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad domain id"})
		return
	}
	var in struct {
		Hostname string `json:"hostname"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	hostname := service.CanonicalHostname(in.Hostname)
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hostname"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant domain update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetTenantDomain(r.Context(), db.GetTenantDomainParams{DomainID: domainID, TargetTenantID: tenantID})
	if err != nil {
		writeNotFound(w)
		return
	}
	domain, err := q.UpdateTenantDomain(r.Context(), db.UpdateTenantDomainParams{
		ID: domainID, TenantID: tenantID, Hostname: hostname,
	})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresUniqueViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "this hostname is already mapped to a tenant"})
			return
		}
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "update tenant domain", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "domain.updated", "domain", domainID.String(),
		"/v1/admin/tenants/{id}/domains/{domainId}", http.StatusOK,
		map[string]string{"tenant_id": tenantID.String(), "previous_hostname": previous.Hostname, "hostname": hostname}); err != nil {
		writeInternalError(w, r, "audit tenant domain update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant domain update", err)
		return
	}
	writeJSON(w, http.StatusOK, domain)
}

// DELETE /v1/admin/tenants/{id}/domains/{domainId}
func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	did, err := parseParam(r, "domainId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad domain id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant domain delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	domain, err := q.GetTenantDomain(r.Context(), db.GetTenantDomainParams{DomainID: did, TargetTenantID: tenantID})
	if err != nil {
		writeNotFound(w)
		return
	}
	n, err := q.RemoveTenantDomain(r.Context(), db.RemoveTenantDomainParams{ID: did, TenantID: tenantID})
	if err != nil {
		writeInternalError(w, r, "delete tenant domain", err)
		return
	}
	if n == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "domain.deleted", "domain", did.String(),
		"/v1/admin/tenants/{id}/domains/{domainId}", http.StatusNoContent,
		map[string]string{"tenant_id": tenantID.String(), "hostname": domain.Hostname}); err != nil {
		writeInternalError(w, r, "audit tenant domain delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant domain delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/tenants/{id}/resources/{resourceId}/status  body {status}
func (s *Server) setResourceStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	rid, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	var in struct{ Status string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "inactive" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource status update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetTenantResource(r.Context(), db.GetTenantResourceParams{ID: rid, TenantID: tenantID})
	if err != nil {
		writeNotFound(w)
		return
	}
	res, err := q.SetTenantResourceStatus(r.Context(), db.SetTenantResourceStatusParams{
		ID: rid, Status: in.Status, TenantID: tenantID,
	})
	if err != nil {
		switch {
		case isNotFound(err):
			writeNotFound(w)
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			// Reactivating may collide with the 1-active-per-type invariant (RN-01).
			writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
		default:
			writeInternalError(w, r, "set tenant resource status", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "resource.status_changed", "resource", rid.String(),
		"/v1/admin/tenants/{id}/resources/{resourceId}/status", http.StatusOK,
		map[string]string{"tenant_id": tenantID.String(), "from": previous.Status, "to": res.Status}); err != nil {
		writeInternalError(w, r, "audit resource status update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource status update", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// DELETE /v1/admin/tenants/{id}/resources/{resourceId}
func (s *Server) deleteResource(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	rid, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	resource, err := q.GetTenantResource(r.Context(), db.GetTenantResourceParams{ID: rid, TenantID: tenantID})
	if err != nil {
		writeNotFound(w)
		return
	}
	n, err := q.DeleteTenantResource(r.Context(), db.DeleteTenantResourceParams{ID: rid, TenantID: tenantID})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresForeignKeyViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "resource_has_linked_dependents"})
			return
		}
		writeInternalError(w, r, "delete tenant resource", err)
		return
	}
	if n == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "resource.deleted", "resource", rid.String(),
		"/v1/admin/tenants/{id}/resources/{resourceId}", http.StatusNoContent,
		map[string]string{"tenant_id": tenantID.String(), "definition_id": resource.ResourceDefinitionID.String()}); err != nil {
		writeInternalError(w, r, "audit resource delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/resource-definitions/{id}/status  body {status}
func (s *Server) setDefinitionStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct{ Status string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "inactive" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin definition status update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetDefinition(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	d, err := q.SetDefinitionStatus(r.Context(), db.SetDefinitionStatusParams{ID: id, Status: in.Status})
	if err != nil {
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "set resource definition status", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.status_changed", "resource_definition", id.String(),
		"/v1/admin/resource-definitions/{id}/status", http.StatusOK,
		map[string]string{"from": previous.Status, "to": d.Status}); err != nil {
		writeInternalError(w, r, "audit definition status update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit definition status update", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// PATCH /v1/admin/resource-definitions/{id} — update the mutable catalog metadata.
// The key is intentionally immutable because consumers and provisioned resources use it as a contract.
func (s *Server) updateDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct{ Name, Description string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin definition update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetDefinition(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	d, err := q.UpdateDefinition(r.Context(), db.UpdateDefinitionParams{ID: id, Name: in.Name, Description: in.Description})
	if err != nil {
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "update resource definition", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.updated", "resource_definition", id.String(),
		"/v1/admin/resource-definitions/{id}", http.StatusOK,
		map[string]any{
			"before": map[string]string{"name": previous.Name, "description": previous.Description},
			"after":  map[string]string{"name": d.Name, "description": d.Description},
		}); err != nil {
		writeInternalError(w, r, "audit definition update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit definition update", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// DELETE /v1/admin/resource-definitions/{id} — delete an unused catalog type.
// PostgreSQL prevents deletion while tenant resources still reference the definition.
func (s *Server) deleteDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin definition delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	d, err := q.GetDefinition(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	n, err := q.DeleteDefinition(r.Context(), id)
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresForeignKeyViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "definition_in_use"})
			return
		}
		writeInternalError(w, r, "delete resource definition", err)
		return
	}
	if n == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.deleted", "resource_definition", id.String(),
		"/v1/admin/resource-definitions/{id}", http.StatusNoContent,
		map[string]string{"key": d.Key, "name": d.Name}); err != nil {
		writeInternalError(w, r, "audit definition delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit definition delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PATCH /v1/admin/resource-definitions/{id}/fields/{fieldId} updates mutable presentation metadata.
// Key, data type, and secret storage remain immutable consumer/storage contracts.
func (s *Server) updateField(w http.ResponseWriter, r *http.Request) {
	definitionID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	fieldID, err := parseParam(r, "fieldId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad field id"})
		return
	}
	var in struct {
		Label    string
		Required bool
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Label = strings.TrimSpace(in.Label)

	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource field metadata update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetResourceField(r.Context(), db.GetResourceFieldParams{
		FieldID: fieldID, ResourceDefinitionID: definitionID,
	})
	if err != nil {
		writeNotFound(w)
		return
	}
	field, err := q.UpdateField(r.Context(), db.UpdateFieldParams{
		FieldID: fieldID, ResourceDefinitionID: definitionID,
		Label: in.Label, Required: in.Required,
	})
	if err != nil {
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "update resource field metadata", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.field_updated", "resource_field", fieldID.String(),
		"/v1/admin/resource-definitions/{id}/fields/{fieldId}", http.StatusOK,
		map[string]any{
			"definition_id": definitionID.String(),
			"field_key":     previous.Key,
			"before": map[string]any{
				"label": previous.Label, "required": previous.Required,
			},
			"after": map[string]any{
				"label": field.Label, "required": field.Required,
			},
		}); err != nil {
		writeInternalError(w, r, "audit resource field metadata update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource field metadata update", err)
		return
	}
	writeJSON(w, http.StatusOK, field)
}

// DELETE /v1/admin/resource-definitions/{id}/fields/{fieldId}
func (s *Server) deleteField(w http.ResponseWriter, r *http.Request) {
	definitionID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	fid, err := parseParam(r, "fieldId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad field id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource field delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	field, err := q.GetResourceField(r.Context(), db.GetResourceFieldParams{FieldID: fid, ResourceDefinitionID: definitionID})
	if err != nil {
		writeNotFound(w)
		return
	}
	n, err := q.RemoveField(r.Context(), db.RemoveFieldParams{
		FieldID: fid, ResourceDefinitionID: definitionID,
	})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresForeignKeyViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "field is in use by tenant resources"})
			return
		}
		writeInternalError(w, r, "delete resource field", err)
		return
	}
	if n == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.field_deleted", "resource_field", fid.String(),
		"/v1/admin/resource-definitions/{id}/fields/{fieldId}", http.StatusNoContent,
		map[string]string{"definition_id": definitionID.String(), "field_key": field.Key}); err != nil {
		writeInternalError(w, r, "audit resource field delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource field delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/admin/overview — aggregated counts + tenant cards.
type overviewTenant struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Status        string `json:"status"`
	PrimaryHost   string `json:"primaryHost"`
	ResourceCount int    `json:"resourceCount"`
}

type overviewResponse struct {
	Tenants           int              `json:"tenants"`
	ActiveTenants     int              `json:"activeTenants"`
	Domains           int              `json:"domains"`
	Resources         int              `json:"resources"`
	Definitions       int              `json:"definitions"`
	ActiveDefinitions int              `json:"activeDefinitions"`
	APIClients        int              `json:"apiClients"`
	TenantCards       []overviewTenant `json:"tenantCards"`
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out := overviewResponse{TenantCards: []overviewTenant{}}

	cards, err := s.Q.ListOverviewTenantCards(ctx)
	if err != nil {
		writeInternalError(w, r, "list overview tenant cards", err)
		return
	}
	out.Tenants = len(cards)
	for _, t := range cards {
		card := overviewTenant{
			ID: t.ID.String(), Name: t.Name, PrimaryHost: t.PrimaryHost,
			ResourceCount: int(t.ResourceCount), Slug: t.Slug, Status: t.Status,
		}
		if t.Status == "active" {
			out.ActiveTenants++
		}
		out.Domains += int(t.DomainCount)
		out.Resources += int(t.ResourceCount)
		out.TenantCards = append(out.TenantCards, card)
	}

	defs, err := s.Q.CountDefinitionsSummary(ctx)
	if err != nil {
		writeInternalError(w, r, "count overview definitions", err)
		return
	}
	out.Definitions = int(defs.Definitions)
	out.ActiveDefinitions = int(defs.ActiveDefinitions)

	clients, err := s.Q.CountAPIClients(ctx)
	if err != nil {
		writeInternalError(w, r, "count overview API clients", err)
		return
	}
	out.APIClients = int(clients)

	writeJSON(w, http.StatusOK, out)
}
