package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func parseID(r *http.Request) (uuid.UUID, error) { return uuid.Parse(chi.URLParam(r, "id")) }

// --- tenants ---

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var in struct{ Slug, Name string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Slug == "" || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug and name required"})
		return
	}
	idempotencyRequest, ok := prepareAdminIdempotency(w, r, "POST /v1/admin/tenants", in, adminMutationTTL)
	if !ok {
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant create", err)
		return
	}
	defer tx.Rollback(r.Context())
	if proceed, _ := s.beginAdminIdempotency(w, r, tx, idempotencyRequest); !proceed {
		return
	}
	q := s.Q.WithTx(tx)
	t, err := q.CreateTenant(r.Context(), db.CreateTenantParams{Slug: in.Slug, Name: in.Name})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresUniqueViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "a tenant with this slug already exists"})
			return
		}
		writeInternalError(w, r, "create tenant", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "tenant.created", "tenant", t.ID.String(),
		"/v1/admin/tenants", http.StatusCreated, map[string]string{"slug": t.Slug, "name": t.Name}); err != nil {
		writeInternalError(w, r, "audit tenant create", err)
		return
	}
	responseBody, err := encodeIdempotentResponse(t)
	if err != nil {
		writeInternalError(w, r, "encode tenant create response", err)
		return
	}
	defer clear(responseBody)
	if err := s.completeAdminIdempotency(r, tx, idempotencyRequest, http.StatusCreated, responseBody); err != nil {
		writeInternalError(w, r, "complete tenant create idempotency", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant create", err)
		return
	}
	writeIdempotentResponse(w, http.StatusCreated, responseBody, false)
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Q.ListTenants(r.Context())
	if err != nil {
		writeInternalError(w, r, "list tenants", err)
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
		if isNotFound(err) {
			writeNotFound(w)
			return
		}
		writeInternalError(w, r, "get tenant", err)
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
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}
	hostname := service.CanonicalHostname(in.Hostname)
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hostname"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin tenant domain create", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	d, err := q.AddTenantDomain(r.Context(), db.AddTenantDomainParams{TenantID: id, Hostname: hostname})
	if err != nil {
		switch {
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "this hostname is already mapped to a tenant"})
		case store.IsPostgresCode(err, store.PostgresForeignKeyViolation):
			writeNotFound(w)
		default:
			writeInternalError(w, r, "add tenant domain", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "domain.added", "domain", d.ID.String(),
		"/v1/admin/tenants/{id}/domains", http.StatusCreated,
		map[string]string{"tenant_id": id.String(), "hostname": hostname}); err != nil {
		writeInternalError(w, r, "audit tenant domain create", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit tenant domain create", err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// --- definitions / fields ---

func (s *Server) createDefinition(w http.ResponseWriter, r *http.Request) {
	var in struct{ Key, Name, Description, Icon string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Key == "" || in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key and name required"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin definition create", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	d, err := q.CreateDefinition(r.Context(), db.CreateDefinitionParams{
		Key: in.Key, Name: in.Name, Description: in.Description, Icon: in.Icon,
	})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresUniqueViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "a definition with this key already exists"})
			return
		}
		writeInternalError(w, r, "create resource definition", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.created", "resource_definition", d.ID.String(),
		"/v1/admin/resource-definitions", http.StatusCreated,
		map[string]string{"key": d.Key, "name": d.Name}); err != nil {
		writeInternalError(w, r, "audit definition create", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit definition create", err)
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
		writeInternalError(w, r, "list resource definitions", err)
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
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}
	if in.DataType == "" {
		in.DataType = "string"
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource field create", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	f, err := q.AddField(r.Context(), db.AddFieldParams{
		ResourceDefinitionID: id, Key: in.Key, Label: in.Label, Hint: in.Hint,
		DataType: in.DataType, Required: in.Required, IsSecret: in.IsSecret, SortOrder: in.SortOrder,
	})
	if err != nil {
		switch {
		case store.IsPostgresCode(err, store.PostgresUniqueViolation):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "a field with this key already exists"})
		case store.IsPostgresCode(err, store.PostgresForeignKeyViolation):
			writeNotFound(w)
		default:
			writeInternalError(w, r, "add resource field", err)
		}
		return
	}
	if err := insertAdminAuditSuccess(r, q, "definition.field_added", "resource_field", f.ID.String(),
		"/v1/admin/resource-definitions/{id}/fields", http.StatusCreated,
		map[string]any{"definition_id": id.String(), "field_key": f.Key, "is_secret": f.IsSecret}); err != nil {
		writeInternalError(w, r, "audit resource field create", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource field create", err)
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
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.DefinitionKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definitionKey required"})
		return
	}
	idempotencyRequest, ok := prepareAdminIdempotency(w, r, "POST /v1/admin/tenants/{id}/resources", struct {
		TenantID uuid.UUID `json:"tenant_id"`
		Input    any       `json:"input"`
	}{TenantID: tenantID, Input: in}, adminMutationTTL)
	if !ok {
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin resource provisioning", err)
		return
	}
	defer tx.Rollback(r.Context())
	if proceed, _ := s.beginAdminIdempotency(w, r, tx, idempotencyRequest); !proceed {
		return
	}
	q := s.Q.WithTx(tx)
	res, err := service.ProvisionResourceInTx(r.Context(), q, s.Cryptor, service.ProvisionResourceInput{
		DefinitionKey: in.DefinitionKey, TenantID: tenantID, Values: in.Values,
	})
	if err != nil {
		writeProvisionError(w, r, err)
		return
	}
	fieldNames := make([]string, 0, len(in.Values))
	for key := range in.Values {
		fieldNames = append(fieldNames, key)
	}
	if err := insertAdminAuditSuccess(r, q, "resource.provisioned", "resource", res.ID.String(),
		"/v1/admin/tenants/{id}/resources", http.StatusCreated,
		map[string]any{"tenant_id": tenantID.String(), "definition_key": in.DefinitionKey, "field_names": fieldNames}); err != nil {
		writeInternalError(w, r, "audit resource provisioning", err)
		return
	}
	responseBody, err := encodeIdempotentResponse(res)
	if err != nil {
		writeInternalError(w, r, "encode resource provisioning response", err)
		return
	}
	defer clear(responseBody)
	if err := s.completeAdminIdempotency(r, tx, idempotencyRequest, http.StatusCreated, responseBody); err != nil {
		writeInternalError(w, r, "complete resource provisioning idempotency", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit resource provisioning", err)
		return
	}
	writeIdempotentResponse(w, http.StatusCreated, responseBody, false)
}

func writeProvisionError(w http.ResponseWriter, r *http.Request, err error) {
	var missing service.MissingRequiredFieldError
	switch {
	case errors.Is(err, service.ErrUnknownDefinition):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown definition"})
	case errors.Is(err, service.ErrUnknownTenant):
		writeNotFound(w)
	case errors.Is(err, service.ErrInactiveDefinition):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definition is inactive"})
	case errors.As(err, &missing):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": missing.Error()})
	case errors.Is(err, service.ErrActiveResourceExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "an active resource of this type already exists"})
	default:
		writeInternalError(w, r, "provision tenant resource", err)
	}
}

// --- api clients ---

func (s *Server) createAPIClient(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string    `json:"name"`
		Scopes    []string  `json:"scopes"`
		RPMLimit  int32     `json:"rpm_limit"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	if err := service.ValidateAPIClientPolicy(s.Now().UTC(), in.Scopes, in.RPMLimit, in.ExpiresAt); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	idempotencyRequest, ok := prepareAdminIdempotency(w, r, "POST /v1/admin/api-clients", in, adminSecretTTL)
	if !ok {
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin API client transaction", err)
		return
	}
	defer tx.Rollback(r.Context())
	if proceed, _ := s.beginAdminIdempotency(w, r, tx, idempotencyRequest); !proceed {
		return
	}
	token, err := service.GenerateAPIToken()
	if err != nil {
		writeInternalError(w, r, "generate API client token", err)
		return
	}
	txq := s.Q.WithTx(tx)
	preview := service.APITokenPreview(token)
	c, err := txq.CreateAPIClient(r.Context(), db.CreateAPIClientParams{
		Name: in.Name, KeyHash: service.HashAPIKey(token), TokenPreview: &preview,
		RpmLimit: &in.RPMLimit, ExpiresAt: pgtype.Timestamptz{Time: in.ExpiresAt, Valid: true},
	})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresUniqueViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "api_client_name_conflict"})
			return
		}
		writeInternalError(w, r, "create API client", err)
		return
	}
	if err := txq.ReplaceAPIClientScopes(r.Context(), db.ReplaceAPIClientScopesParams{
		ApiClientID: c.ID, Scopes: in.Scopes,
	}); err != nil {
		writeInternalError(w, r, "set API client scopes", err)
		return
	}
	if err := insertAdminAuditSuccess(
		r, txq, "api_client.created", "api_client", c.ID.String(),
		"/v1/admin/api-clients", http.StatusCreated, map[string]any{
			"name": c.Name, "scopes": in.Scopes, "rpm_limit": in.RPMLimit,
			"expires_at": in.ExpiresAt.UTC().Format(time.RFC3339),
		},
	); err != nil {
		writeInternalError(w, r, "audit API client creation", err)
		return
	}
	responseBody, err := encodeIdempotentResponse(createAPIClientResponse{
		Client: newAPIClientView(c, in.Scopes, s.Now().UTC()),
		Token:  token,
	})
	if err != nil {
		writeInternalError(w, r, "encode API client response", err)
		return
	}
	defer clear(responseBody)
	if err := s.completeAdminIdempotency(r, tx, idempotencyRequest, http.StatusCreated, responseBody); err != nil {
		writeInternalError(w, r, "complete API client idempotency", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit API client creation", err)
		return
	}
	writeIdempotentResponse(w, http.StatusCreated, responseBody, true)
}

func (s *Server) listAPIClients(w http.ResponseWriter, r *http.Request) {
	cs, err := s.Q.ListAPIClients(r.Context())
	if err != nil {
		writeInternalError(w, r, "list API clients", err)
		return
	}
	out := make([]apiClientView, 0, len(cs))
	for _, client := range cs {
		scopes, err := s.Q.ListAPIClientScopes(r.Context(), client.ID)
		if err != nil {
			writeInternalError(w, r, "list API client scopes", err)
			return
		}
		out = append(out, newAPIClientView(client, scopes, s.Now().UTC()))
	}
	writeJSON(w, http.StatusOK, out)
}
