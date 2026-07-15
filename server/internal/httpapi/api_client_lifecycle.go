package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store"
	"github.com/djalmajr/tenancit/server/internal/store/db"
)

type apiClientPolicyInput struct {
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	RPMLimit  int32     `json:"rpm_limit"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Server) updateAPIClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in apiClientPolicyInput
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
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin API client policy update", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	previous, err := q.GetAPIClient(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	if previous.Status != "active" || !s.Now().UTC().Before(previous.ExpiresAt) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "client_must_rotate"})
		return
	}
	updated, err := q.UpdateAPIClientPolicy(r.Context(), db.UpdateAPIClientPolicyParams{
		ID: id, Name: in.Name, RpmLimit: in.RPMLimit,
		ExpiresAt: in.ExpiresAt,
	})
	if err != nil {
		if store.IsPostgresCode(err, store.PostgresUniqueViolation) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "api_client_name_conflict"})
			return
		}
		writeInternalError(w, r, "update API client policy", err)
		return
	}
	if err := q.ReplaceAPIClientScopes(r.Context(), db.ReplaceAPIClientScopesParams{ApiClientID: id, Scopes: in.Scopes}); err != nil {
		writeInternalError(w, r, "replace API client scopes", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "api_client.policy_updated", "api_client", id.String(),
		"/v1/admin/api-clients/{id}", http.StatusOK, map[string]any{
			"before": map[string]any{"name": previous.Name, "rpm_limit": previous.RpmLimit, "expires_at": previous.ExpiresAt},
			"after":  map[string]any{"name": in.Name, "scopes": in.Scopes, "rpm_limit": in.RPMLimit, "expires_at": in.ExpiresAt.UTC().Format(time.RFC3339)},
		}); err != nil {
		writeInternalError(w, r, "audit API client policy update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit API client policy update", err)
		return
	}
	writeJSON(w, http.StatusOK, newAPIClientView(updated, in.Scopes, s.Now().UTC()))
}

func (s *Server) rotateAPIClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	var in struct {
		GraceSeconds int32 `json:"grace_seconds"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.GraceSeconds < 0 || in.GraceSeconds > 86400 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_rotation_grace"})
		return
	}
	idempotencyRequest, ok := prepareAdminIdempotency(w, r, "POST /v1/admin/api-clients/{id}/rotate", struct {
		ClientID any `json:"client_id"`
		Input    any `json:"input"`
	}{ClientID: id, Input: in}, adminSecretTTL)
	if !ok {
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin API client rotation", err)
		return
	}
	defer tx.Rollback(r.Context())
	if proceed, _ := s.beginAdminIdempotency(w, r, tx, idempotencyRequest); !proceed {
		return
	}
	token, err := service.GenerateAPIToken()
	if err != nil {
		writeInternalError(w, r, "generate rotated API client token", err)
		return
	}
	q := s.Q.WithTx(tx)
	current, err := q.GetAPIClient(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	if current.Status != "active" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "client_must_rotate"})
		return
	}
	if in.GraceSeconds > 0 {
		if err := q.SavePreviousAPIClientToken(r.Context(), db.SavePreviousAPIClientTokenParams{
			ApiClientID: id, KeyHash: current.KeyHash,
			ValidUntil: s.Now().UTC().Add(time.Duration(in.GraceSeconds) * time.Second),
		}); err != nil {
			writeInternalError(w, r, "save previous API client token", err)
			return
		}
	}
	preview := service.APITokenPreview(token)
	rotated, err := q.RotateAPIClientToken(r.Context(), db.RotateAPIClientTokenParams{
		ID: id, KeyHash: service.HashAPIKey(token), TokenPreview: preview,
	})
	if err != nil {
		writeInternalError(w, r, "rotate API client token", err)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "api_client.rotated", "api_client", id.String(),
		"/v1/admin/api-clients/{id}/rotate", http.StatusCreated,
		map[string]int32{"grace_seconds": in.GraceSeconds}); err != nil {
		writeInternalError(w, r, "audit API client rotation", err)
		return
	}
	scopes, err := q.ListAPIClientScopes(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, "list rotated API client scopes", err)
		return
	}
	responseBody, err := encodeIdempotentResponse(createAPIClientResponse{Client: newAPIClientView(rotated, scopes, s.Now().UTC()), Token: token})
	if err != nil {
		writeInternalError(w, r, "encode rotated API client response", err)
		return
	}
	defer clear(responseBody)
	if err := s.completeAdminIdempotency(r, tx, idempotencyRequest, http.StatusCreated, responseBody); err != nil {
		writeInternalError(w, r, "complete API client rotation idempotency", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit API client rotation", err)
		return
	}
	writeIdempotentResponse(w, http.StatusCreated, responseBody, true)
}

func (s *Server) revokeAPIClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin API client revocation", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	current, err := q.GetAPIClient(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	revoked, err := q.SetAPIClientStatus(r.Context(), db.SetAPIClientStatusParams{ID: id, Status: "revoked"})
	if err != nil {
		writeInternalError(w, r, "revoke API client", err)
		return
	}
	if current.Status != "revoked" {
		if err := insertAdminAuditSuccess(r, q, "api_client.revoked", "api_client", id.String(),
			"/v1/admin/api-clients/{id}/revoke", http.StatusOK, map[string]string{"from": current.Status, "to": "revoked"}); err != nil {
			writeInternalError(w, r, "audit API client revocation", err)
			return
		}
	}
	scopes, err := q.ListAPIClientScopes(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, "list revoked API client scopes", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit API client revocation", err)
		return
	}
	writeJSON(w, http.StatusOK, newAPIClientView(revoked, scopes, s.Now().UTC()))
}

func (s *Server) deleteAPIClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin API client delete", err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Q.WithTx(tx)
	current, err := q.GetAPIClient(r.Context(), id)
	if err != nil {
		writeNotFound(w)
		return
	}
	if current.Status != "revoked" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "client_not_revoked"})
		return
	}
	deleted, err := q.DeleteAPIClient(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, "delete API client", err)
		return
	}
	if deleted == 0 {
		writeNotFound(w)
		return
	}
	if err := insertAdminAuditSuccess(r, q, "api_client.deleted", "api_client", id.String(),
		"/v1/admin/api-clients/{id}", http.StatusNoContent, map[string]string{"name": current.Name}); err != nil {
		writeInternalError(w, r, "audit API client delete", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit API client delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
