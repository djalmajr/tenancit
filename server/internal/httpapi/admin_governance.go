package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	appsettings "github.com/djalmajr/tenancit/server/internal/settings"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type settingsUpdateBody struct {
	Values map[string]string `json:"values"`
}

type settingsAuditMetadata struct {
	ChangedKeys []string `json:"changed_keys"`
	Revision    int64    `json:"revision"`
}

func settingsETag(revision int64) string { return fmt.Sprintf(`"settings-%d"`, revision) }

func parseSettingsETag(value string) (int64, error) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, `"`)
	if !strings.HasPrefix(value, "settings-") {
		return 0, errors.New("invalid settings ETag")
	}
	return strconv.ParseInt(strings.TrimPrefix(value, "settings-"), 10, 64)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.Settings.Get(r.Context())
	if err != nil {
		writeInternalError(w, r, "get settings", err)
		return
	}
	w.Header().Set("ETag", settingsETag(snapshot.Revision))
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	expectedRevision, err := parseSettingsETag(r.Header.Get("If-Match"))
	if err != nil {
		writeJSON(w, http.StatusPreconditionRequired, map[string]string{"error": "settings_if_match_required"})
		return
	}
	var body settingsUpdateBody
	if !decodeJSON(w, r, &body) {
		return
	}
	actor, ok := principalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin settings update", err)
		return
	}
	defer tx.Rollback(r.Context())
	snapshot, err := s.Settings.Update(r.Context(), tx, expectedRevision, body.Values, appsettings.Actor{
		Kind: string(actor.Kind), Issuer: actor.Issuer, Subject: actor.Subject,
	})
	if errors.Is(err, appsettings.ErrRevisionConflict) {
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": "settings_revision_conflict"})
		return
	}
	if errors.Is(err, appsettings.ErrInvalidSettings) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_settings"})
		return
	}
	if err != nil {
		writeInternalError(w, r, "update settings", err)
		return
	}
	keys := make([]string, 0, len(body.Values))
	for key := range body.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if err := insertAdminAuditSuccess(r, db.New(tx), "settings.updated", "settings", "global", "/v1/admin/settings", http.StatusOK, settingsAuditMetadata{
		ChangedKeys: keys, Revision: snapshot.Revision,
	}); err != nil {
		writeInternalError(w, r, "audit settings update", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit settings update", err)
		return
	}
	w.Header().Set("ETag", settingsETag(snapshot.Revision))
	writeJSON(w, http.StatusOK, snapshot)
}

type adminSessionView struct {
	ID            string     `json:"id"`
	Issuer        string     `json:"issuer"`
	Subject       string     `json:"subject"`
	Label         string     `json:"label"`
	Roles         []string   `json:"roles"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsedAt    time.Time  `json:"last_used_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	IdleExpiresAt time.Time  `json:"idle_expires_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	Status        string     `json:"status"`
	Current       bool       `json:"current"`
}

func effectiveAdminSessionStatus(session adminauth.GovernedSession, now time.Time) string {
	if session.RevokedAt.Valid {
		return "revoked"
	}
	if !now.Before(session.ExpiresAt) {
		return "expired"
	}
	if !now.Before(session.IdleExpiresAt) {
		return "idle_expired"
	}
	return "active"
}

func (s *Server) listAdminSessions(w http.ResponseWriter, r *http.Request) {
	if s.AdminAuthStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session_governance_unavailable"})
		return
	}
	actor, _ := principalFromContext(r.Context())
	sessions, err := s.AdminAuthStore.ListAdminSessions(r.Context())
	if err != nil {
		writeInternalError(w, r, "list admin sessions", err)
		return
	}
	now := s.Now().UTC()
	views := make([]adminSessionView, 0, len(sessions))
	for _, session := range sessions {
		var revokedAt *time.Time
		if session.RevokedAt.Valid {
			value := session.RevokedAt.Time
			revokedAt = &value
		}
		views = append(views, adminSessionView{
			ID: session.ID, Issuer: session.Issuer, Subject: session.Subject, Label: session.Label,
			Roles: session.Roles, CreatedAt: session.CreatedAt, LastUsedAt: session.LastUsedAt,
			ExpiresAt: session.ExpiresAt, IdleExpiresAt: session.IdleExpiresAt, RevokedAt: revokedAt,
			Status: effectiveAdminSessionStatus(session, now), Current: session.ID == actor.SessionID,
		})
	}
	writeJSON(w, http.StatusOK, views)
}

type sessionRevokeAuditMetadata struct {
	Mode  string `json:"mode"`
	Count int64  `json:"count"`
}

func (s *Server) revokeAdminSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	actor, ok := principalFromContext(r.Context())
	currentID, currentErr := uuid.Parse(actor.SessionID)
	if err != nil || !ok || currentErr != nil || s.AdminAuthStore == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_admin_session"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin admin session revoke", err)
		return
	}
	defer tx.Rollback(r.Context())
	err = s.AdminAuthStore.RevokeOtherAdminSession(r.Context(), tx, id, currentID, s.Now().UTC())
	if errors.Is(err, adminauth.ErrCurrentSessionRequiresLogout) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "current_session_requires_logout"})
		return
	}
	if errors.Is(err, adminauth.ErrSessionNotActive) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session_not_active"})
		return
	}
	if err != nil {
		writeInternalError(w, r, "revoke admin session", err)
		return
	}
	if err := insertAdminAuditSuccess(r, db.New(tx), "admin_session.revoked", "admin_session", id.String(), "/v1/admin/sessions/{id}", http.StatusNoContent, sessionRevokeAuditMetadata{Mode: "single", Count: 1}); err != nil {
		writeInternalError(w, r, "audit admin session revoke", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit admin session revoke", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type revokePrincipalBody struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

func (s *Server) revokeAdminPrincipalSessions(w http.ResponseWriter, r *http.Request) {
	var body revokePrincipalBody
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Issuer, body.Subject = strings.TrimSpace(body.Issuer), strings.TrimSpace(body.Subject)
	actor, ok := principalFromContext(r.Context())
	currentID, err := uuid.Parse(actor.SessionID)
	if !ok || err != nil || body.Issuer == "" || body.Subject == "" || s.AdminAuthStore == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_admin_principal"})
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		writeInternalError(w, r, "begin principal session revoke", err)
		return
	}
	defer tx.Rollback(r.Context())
	count, err := s.AdminAuthStore.RevokePrincipalSessions(r.Context(), tx, body.Issuer, body.Subject, currentID, s.Now().UTC())
	if err != nil {
		writeInternalError(w, r, "revoke principal sessions", err)
		return
	}
	if err := insertAdminAuditSuccess(r, db.New(tx), "admin_session.principal_revoked", "admin_principal", body.Subject, "/v1/admin/sessions/revoke-principal", http.StatusOK, sessionRevokeAuditMetadata{Mode: "principal", Count: count}); err != nil {
		writeInternalError(w, r, "audit principal session revoke", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeInternalError(w, r, "commit principal session revoke", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revoked": count})
}
