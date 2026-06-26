package httpapi

import (
	"net/http"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
)

// listTenantDomains: GET /v1/admin/tenants/{id}/domains
func (s *Server) listTenantDomains(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	domains, err := s.Q.ListTenantDomains(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if domains == nil {
		domains = []db.TenantDomain{}
	}
	writeJSON(w, http.StatusOK, domains)
}

type adminResource struct {
	ID            string                       `json:"id"`
	DefinitionKey string                       `json:"definitionKey"`
	DefinitionID  string                       `json:"definitionId"`
	Name          string                       `json:"name"`
	Status        string                       `json:"status"`
	Fields        []service.ResourceFieldValue `json:"fields"`
}

// listTenantResources: GET /v1/admin/tenants/{id}/resources?reveal=true
// Returns each active/inactive resource with its definition fields and values,
// masking secrets unless reveal=true (RN-06).
func (s *Server) listTenantResources(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	reveal := r.URL.Query().Get("reveal") == "true"
	ctx := r.Context()

	resources, err := s.Q.ListTenantResources(ctx, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	out := make([]adminResource, 0, len(resources))
	for _, res := range resources {
		built, err := service.BuildResourceFields(ctx, service.BuildResourceFieldsDeps{
			Cryptor: s.Cryptor, Queries: s.Q,
		}, service.BuildResourceFieldsInput{Resource: res, Reveal: reveal})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		ar := adminResource{
			ID:            res.ID.String(),
			DefinitionKey: built.Definition.Key,
			DefinitionID:  built.Definition.ID.String(),
			Name:          built.Definition.Name,
			Status:        res.Status,
			Fields:        built.Fields,
		}
		out = append(out, ar)
	}
	writeJSON(w, http.StatusOK, out)
}

type adminDefinitionDetail struct {
	Definition db.ResourceDefinition `json:"definition"`
	Fields     []db.ResourceField    `json:"fields"`
}

// getDefinition: GET /v1/admin/resource-definitions/{id}
func (s *Server) getDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	ctx := r.Context()
	def, err := s.Q.GetDefinition(ctx, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	fields, err := s.Q.ListFields(ctx, def.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if fields == nil {
		fields = []db.ResourceField{}
	}
	writeJSON(w, http.StatusOK, adminDefinitionDetail{Definition: def, Fields: fields})
}
