package httpapi

import (
	"net/http"

	"github.com/djalmajr/konvario/server/internal/store/db"
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
	if err := decode(r, &in); err != nil || in.Name == "" || in.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
		return
	}
	if in.Status == "" {
		in.Status = "active"
	}
	t, err := s.Q.UpdateTenant(r.Context(), db.UpdateTenantParams{
		ID: id, Name: in.Name, Slug: in.Slug, Status: in.Status,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// DELETE /v1/admin/tenants/{id}/domains/{domainId}
func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	did, err := parseParam(r, "domainId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad domain id"})
		return
	}
	if err := s.Q.RemoveTenantDomain(r.Context(), did); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/tenants/{id}/resources/{resourceId}/status  body {status}
func (s *Server) setResourceStatus(w http.ResponseWriter, r *http.Request) {
	rid, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	var in struct{ Status string }
	if err := decode(r, &in); err != nil || (in.Status != "active" && in.Status != "inactive") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
		return
	}
	res, err := s.Q.SetTenantResourceStatus(r.Context(), db.SetTenantResourceStatusParams{ID: rid, Status: in.Status})
	if err != nil {
		// reactivating may collide with the 1-active-per-type invariant (RN-01)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// DELETE /v1/admin/tenants/{id}/resources/{resourceId}
func (s *Server) deleteResource(w http.ResponseWriter, r *http.Request) {
	rid, err := parseParam(r, "resourceId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad resource id"})
		return
	}
	if err := s.Q.DeleteTenantResource(r.Context(), rid); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
	if err := decode(r, &in); err != nil || (in.Status != "active" && in.Status != "inactive") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|inactive"})
		return
	}
	d, err := s.Q.SetDefinitionStatus(r.Context(), db.SetDefinitionStatusParams{ID: id, Status: in.Status})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// DELETE /v1/admin/resource-definitions/{id}/fields/{fieldId}
func (s *Server) deleteField(w http.ResponseWriter, r *http.Request) {
	fid, err := parseParam(r, "fieldId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad field id"})
		return
	}
	if err := s.Q.RemoveField(r.Context(), fid); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /v1/admin/api-clients/{id}/status  body {status}
func (s *Server) setAPIClientStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct{ Status string }
	if err := decode(r, &in); err != nil || (in.Status != "active" && in.Status != "revoked") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be active|revoked"})
		return
	}
	c, err := s.Q.SetAPIClientStatus(r.Context(), db.SetAPIClientStatusParams{ID: id, Status: in.Status})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, c)
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out.Definitions = int(defs.Definitions)
	out.ActiveDefinitions = int(defs.ActiveDefinitions)

	clients, err := s.Q.CountAPIClients(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out.APIClients = int(clients)

	writeJSON(w, http.StatusOK, out)
}
