package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/djalmajr/konvario/server/internal/service"
	"github.com/djalmajr/konvario/server/internal/store/db"
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
	ds, err := s.Q.ListDefinitionsWithCounts(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]definitionListItem, 0, len(ds))
	for _, d := range ds {
		item := definitionListItem{
			ResourceDefinition: db.ResourceDefinition{
				ID: d.ID, Key: d.Key, Name: d.Name, Description: d.Description, Icon: d.Icon,
				Status: d.Status, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
			},
			FieldCount:  int(d.FieldCount),
			SecretCount: int(d.SecretCount),
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
	res, err := service.ProvisionResource(r.Context(), service.ProvisionResourceDeps{
		Cryptor: s.Cryptor, Queries: s.Q, TxStarter: s.DB,
	}, service.ProvisionResourceInput{
		DefinitionKey: in.DefinitionKey, TenantID: tenantID, Values: in.Values,
	})
	if err != nil {
		writeProvisionError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func writeProvisionError(w http.ResponseWriter, err error) {
	var missing service.MissingRequiredFieldError
	switch {
	case errors.Is(err, service.ErrUnknownDefinition):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown definition"})
	case errors.Is(err, service.ErrInactiveDefinition):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definition is inactive"})
	case errors.As(err, &missing):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": missing.Error()})
	case errors.Is(err, service.ErrActiveResourceExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
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
