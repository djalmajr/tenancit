package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/centralit/resource-tenant/server/internal/service"
	"github.com/centralit/resource-tenant/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

func parseID(r *http.Request) (uuid.UUID, error) { return uuid.Parse(chi.URLParam(r, "id")) }

// --- tenants ---

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var in struct{ Slug, Name string }
	if err := decode(r, &in); err != nil || in.Slug == "" || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug and name required"})
		return
	}
	t, err := s.Q.CreateTenant(r.Context(), db.CreateTenantParams{Slug: in.Slug, Name: in.Name})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Q.ListTenants(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	t, err := s.Q.GetTenant(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) addDomain(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct{ Hostname string }
	if err := decode(r, &in); err != nil || in.Hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}
	d, err := s.Q.AddTenantDomain(r.Context(), db.AddTenantDomainParams{TenantID: id, Hostname: in.Hostname})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// --- definitions / fields ---

func (s *Server) createDefinition(w http.ResponseWriter, r *http.Request) {
	var in struct{ Key, Name, Description, Icon string }
	if err := decode(r, &in); err != nil || in.Key == "" || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key and name required"})
		return
	}
	d, err := s.Q.CreateDefinition(r.Context(), db.CreateDefinitionParams{
		Key: in.Key, Name: in.Name, Description: in.Description, Icon: in.Icon,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

type definitionListItem struct {
	db.ResourceDefinition
	FieldCount  int `json:"fieldCount"`
	SecretCount int `json:"secretCount"`
}

func (s *Server) listDefinitions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ds, err := s.Q.ListDefinitions(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]definitionListItem, 0, len(ds))
	for _, d := range ds {
		item := definitionListItem{ResourceDefinition: d}
		fields, _ := s.Q.ListFields(ctx, d.ID)
		item.FieldCount = len(fields)
		for _, f := range fields {
			if f.IsSecret {
				item.SecretCount++
			}
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) addField(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct {
		Key, Label, Hint, DataType string
		Required, IsSecret         bool
		SortOrder                  int32
	}
	if err := decode(r, &in); err != nil || in.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}
	if in.DataType == "" {
		in.DataType = "string"
	}
	f, err := s.Q.AddField(r.Context(), db.AddFieldParams{
		ResourceDefinitionID: id, Key: in.Key, Label: in.Label, Hint: in.Hint,
		DataType: in.DataType, Required: in.Required, IsSecret: in.IsSecret, SortOrder: in.SortOrder,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// --- tenant resources (validates + encrypts) ---

func (s *Server) createResource(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct {
		DefinitionKey string            `json:"definitionKey"`
		Values        map[string]string `json:"values"`
	}
	if err := decode(r, &in); err != nil || in.DefinitionKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definitionKey required"})
		return
	}
	ctx := r.Context()

	def, err := s.Q.GetDefinitionByKey(ctx, in.DefinitionKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown definition"})
		return
	}
	if def.Status != "active" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definition is inactive"}) // RN-08
		return
	}
	fields, err := s.Q.ListFields(ctx, def.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, f := range fields { // RN-03
		if f.Required {
			if v, ok := in.Values[f.Key]; !ok || v == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required field: " + f.Key})
				return
			}
		}
	}

	res, err := s.Q.CreateTenantResource(ctx, db.CreateTenantResourceParams{
		TenantID: tenantID, ResourceDefinitionID: def.ID,
	})
	if err != nil { // RN-01 unique violation surfaces here
		writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
		return
	}

	for _, f := range fields {
		raw, ok := in.Values[f.Key]
		if !ok {
			continue
		}
		p, err := service.EncodeValueFor(s.Cryptor, f.IsSecret, raw) // RN-04
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		p.TenantResourceID = res.ID
		p.ResourceFieldID = f.ID
		if _, err := s.Q.UpsertResourceValue(ctx, p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusCreated, res)
}

// --- api clients ---

func (s *Server) createAPIClient(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name string }
	if err := decode(r, &in); err != nil || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	token, err := service.GenerateAPIToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	c, err := s.Q.CreateAPIClient(r.Context(), db.CreateAPIClientParams{
		Name: in.Name, KeyHash: service.HashAPIKey(token),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"client": c, "token": token}) // token shown once
}

func (s *Server) listAPIClients(w http.ResponseWriter, r *http.Request) {
	cs, err := s.Q.ListAPIClients(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cs)
}
