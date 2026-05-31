package httpapi

import (
	"net/http"

	"github.com/centralit/resource-tenant/server/internal/service"
	"github.com/centralit/resource-tenant/server/internal/store/db"
	"github.com/google/uuid"
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

type adminFieldValue struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	DataType string `json:"dataType"`
	Required bool   `json:"required"`
	IsSecret bool   `json:"isSecret"`
	Value    string `json:"value"`
}

type adminResource struct {
	ID            string            `json:"id"`
	DefinitionKey string            `json:"definitionKey"`
	DefinitionID  string            `json:"definitionId"`
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	Fields        []adminFieldValue `json:"fields"`
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
		def, err := s.Q.GetDefinition(ctx, res.ResourceDefinitionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		fields, err := s.Q.ListFields(ctx, def.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		values, err := s.Q.ListResourceValues(ctx, res.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		valueByField := make(map[uuid.UUID]db.TenantResourceValue, len(values))
		for _, v := range values {
			valueByField[v.ResourceFieldID] = v
		}

		ar := adminResource{
			ID:            res.ID.String(),
			DefinitionKey: def.Key,
			DefinitionID:  def.ID.String(),
			Name:          def.Name,
			Status:        res.Status,
		}
		for _, f := range fields {
			fv := adminFieldValue{
				Key:      f.Key,
				Label:    f.Label,
				DataType: f.DataType,
				Required: f.Required,
				IsSecret: f.IsSecret,
			}
			if v, ok := valueByField[f.ID]; ok {
				shown, err := service.PresentValue(s.Cryptor, f.IsSecret, v, reveal)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				fv.Value = shown
			}
			ar.Fields = append(ar.Fields, fv)
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
