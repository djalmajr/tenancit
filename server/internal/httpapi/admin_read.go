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
		writeInternalError(w, r, "list tenant domains", err)
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
	if reveal {
		w.Header().Set("Cache-Control", secretResponseCacheControl)
	}
	ctx := r.Context()

	headers, err := service.LoadResourceHeaders(ctx, s.Q, id, true)
	if err != nil {
		writeInternalError(w, r, "list tenant resource headers", err)
		return
	}
	built, err := service.BuildResourceFieldsBatch(ctx, s.Q, s.Cryptor, headers, reveal)
	if err != nil {
		writeInternalError(w, r, "assemble tenant resources", err)
		return
	}
	if reveal {
		resourceIDs := make([]string, 0, len(built))
		secretKeys := make([]string, 0)
		seenSecretKeys := make(map[string]struct{})
		for _, resource := range built {
			resourceIDs = append(resourceIDs, resource.Header.Resource.ID.String())
			for _, field := range resource.Fields {
				if !field.IsSecret {
					continue
				}
				if _, exists := seenSecretKeys[field.Key]; exists {
					continue
				}
				seenSecretKeys[field.Key] = struct{}{}
				secretKeys = append(secretKeys, field.Key)
			}
		}
		tx, err := s.DB.Begin(ctx)
		if err != nil {
			writeInternalError(w, r, "begin secret reveal audit", err)
			return
		}
		defer tx.Rollback(ctx)
		if err := insertAdminAuditSuccess(
			r, s.Q.WithTx(tx), "secret.revealed", "tenant", id.String(),
			"/v1/admin/tenants/{id}/resources", http.StatusOK,
			map[string]any{
				"resource_ids": resourceIDs, "secret_field_keys": secretKeys,
				"resource_count": len(resourceIDs), "secret_field_count": len(secretKeys),
			},
		); err != nil {
			writeInternalError(w, r, "audit secret reveal", err)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeInternalError(w, r, "commit secret reveal audit", err)
			return
		}
	}

	out := make([]adminResource, 0, len(built))
	for _, resource := range built {
		header := resource.Header
		ar := adminResource{
			ID:            header.Resource.ID.String(),
			DefinitionKey: header.DefinitionKey,
			DefinitionID:  header.Resource.ResourceDefinitionID.String(),
			Name:          header.DefinitionName,
			Status:        header.Resource.Status,
			Fields:        resource.Fields,
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
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "get resource definition", err)
		return
	}
	fields, err := s.Q.ListFields(ctx, def.ID)
	if err != nil {
		writeInternalError(w, r, "list resource definition fields", err)
		return
	}
	if fields == nil {
		fields = []db.ResourceField{}
	}
	writeJSON(w, http.StatusOK, adminDefinitionDetail{Definition: def, Fields: fields})
}
