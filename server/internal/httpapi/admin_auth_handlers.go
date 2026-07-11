package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/go-chi/chi/v5/middleware"
)

type AdminAuthRuntime struct {
	Config   adminauth.Config
	OIDC     *adminauth.OIDCManager
	Sessions *adminauth.SessionManager
}

func (s *Server) getAdminAuthConfig(w http.ResponseWriter, _ *http.Request) {
	mode := adminauth.ModeLegacySharedToken
	if s.AdminAuth != nil {
		mode = s.AdminAuth.Config.Mode
	}
	response := map[string]any{"mode": mode}
	if mode == adminauth.ModeOIDC {
		response["login_url"] = "/v1/auth/login"
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) startOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.AdminAuth == nil || s.AdminAuth.OIDC == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oidc_unavailable"})
		return
	}
	start, err := s.AdminAuth.OIDC.Start(r.Context(), r.URL.Query().Get("return_to"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_login_request"})
		return
	}
	http.Redirect(w, r, start.AuthorizationURL, http.StatusFound)
}

func (s *Server) completeOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.AdminAuth == nil || s.AdminAuth.OIDC == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oidc_unavailable"})
		return
	}
	completed, err := s.AdminAuth.OIDC.Complete(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
	if err != nil {
		_ = s.insertAuthenticationAudit(r, adminauth.SessionIdentity{}, "admin.login_failed", "denied", http.StatusUnauthorized, adminauth.FailureStage(err))
		slog.Warn("OIDC callback rejected", "request_id", middleware.GetReqID(r.Context()), "failure_stage", adminauth.FailureStage(err), "provider_failure_stage", adminauth.ProviderFailureStage(err))
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "oidc_callback_invalid"})
		return
	}
	if err := s.insertAuthenticationAudit(r, completed.Identity, "admin.login_succeeded", "success", http.StatusSeeOther, ""); err != nil {
		_ = s.AdminAuth.Sessions.Revoke(r.Context(), completed.Identity.SessionID)
		writeInternalError(w, r, "audit OIDC login", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: s.AdminAuth.Config.CookieName, Value: completed.Token,
		Path: "/", HttpOnly: true, Secure: s.AdminAuth.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode, Expires: completed.Identity.ExpiresAt,
	})
	http.Redirect(w, r, completed.RedirectAfter, http.StatusSeeOther)
}

func (s *Server) getAdminSession(w http.ResponseWriter, r *http.Request) {
	value, ok := principalFromContext(r.Context())
	if !ok || value.Kind != principalKindOIDCUser || s.AdminAuth == nil || s.AdminAuth.Sessions == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	cookie, err := r.Cookie(s.AdminAuth.Config.CookieName)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	identity, err := s.AdminAuth.Sessions.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind": string(value.Kind), "issuer": value.Issuer, "subject": value.Subject,
		"label": value.Label, "session_id": value.SessionID, "roles": value.Roles,
		"permissions": identity.Permissions,
		"csrf_token":  identity.CSRFToken, "expires_at": identity.ExpiresAt,
		"idle_expires_at": identity.IdleExpiresAt,
	})
}

func (s *Server) logoutAdminSession(w http.ResponseWriter, r *http.Request) {
	value, ok := principalFromContext(r.Context())
	if !ok || s.AdminAuth == nil || s.AdminAuth.Sessions == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	if err := s.AdminAuth.Sessions.Revoke(r.Context(), value.SessionID); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
		return
	}
	identity := adminauth.SessionIdentity{Issuer: value.Issuer, Subject: value.Subject, Label: value.Label, SessionID: value.SessionID}
	if err := s.insertAuthenticationAudit(r, identity, "admin.logout_succeeded", "success", http.StatusNoContent, ""); err != nil {
		writeInternalError(w, r, "audit OIDC logout", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: s.AdminAuth.Config.CookieName, Value: "", Path: "/",
		HttpOnly: true, Secure: s.AdminAuth.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}

type authenticationAuditMetadata struct {
	FailureStage string `json:"failure_stage,omitempty"`
}

func (s *Server) insertAuthenticationAudit(r *http.Request, identity adminauth.SessionIdentity, action, result string, status int, failureStage string) error {
	metadata, err := json.Marshal(authenticationAuditMetadata{FailureStage: failureStage})
	if err != nil {
		return err
	}
	actorKind, actorSubject := "unauthenticated", "anonymous"
	var issuer, label *string
	if identity.Subject != "" {
		actorKind, actorSubject = "oidc_user", identity.Subject
		issuer, label = optionalString(identity.Issuer), optionalString(identity.Label)
	}
	routeTemplate := "/v1/auth/callback"
	targetID := identity.SessionID
	if targetID == "" {
		targetID = "login"
	}
	if action == "admin.logout_succeeded" {
		routeTemplate = "/v1/auth/logout"
	}
	_, err = s.Q.InsertAdminAuditEvent(r.Context(), db.InsertAdminAuditEventParams{
		RequestID: middleware.GetReqID(r.Context()), ActorKind: actorKind, ActorIssuer: issuer,
		ActorSubject: actorSubject, ActorLabel: label, Action: action, TargetType: "admin_session",
		TargetID: targetID, Result: result, HttpMethod: r.Method,
		RouteTemplate: routeTemplate, HttpStatus: int16(status), Metadata: metadata,
	})
	return err
}
