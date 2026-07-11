package httpapi

import (
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func parseParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
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
	if in.Name == "" || in.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
		return
	}
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
