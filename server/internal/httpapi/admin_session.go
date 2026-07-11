package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
)

type sessionAuthenticator interface {
	Authenticate(context.Context, string) (adminauth.SessionIdentity, error)
}

func RequireAdminSession(authenticator sessionAuthenticator, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator == nil || cookieName == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
				return
			}
			cookie, err := r.Cookie(cookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
				return
			}
			identity, err := authenticator.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_auth_required"})
				return
			}
			value, err := principalFromSessionIdentity(identity)
			if err != nil {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin_role_invalid"})
				return
			}
			next.ServeHTTP(w, r.WithContext(contextWithPrincipal(r.Context(), value)))
		})
	}
}

func principalFromSessionIdentity(identity adminauth.SessionIdentity) (principal, error) {
	roles := make([]adminRole, 0, len(identity.Roles))
	for _, role := range identity.Roles {
		switch role {
		case adminauth.RoleViewer:
			roles = append(roles, roleViewer)
		case adminauth.RoleOperator:
			roles = append(roles, roleOperator)
		case adminauth.RoleSecurityAdmin:
			roles = append(roles, roleSecurityAdmin)
		default:
			return principal{}, errors.New("unknown admin session role")
		}
	}
	value, err := newOIDCPrincipal(identity.Issuer, identity.Subject, identity.Label, identity.SessionID, roles)
	if err != nil {
		return principal{}, err
	}
	value.csrfTokenHash = identity.CSRFTokenHash
	return value, nil
}

func RequireAdminCSRF(adminOrigin string) func(http.Handler) http.Handler {
	adminOrigin = strings.TrimSuffix(adminOrigin, "/")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresCSRF(r) {
				next.ServeHTTP(w, r)
				return
			}
			value, ok := principalFromContext(r.Context())
			if !ok || value.Kind != principalKindOIDCUser || !sameRequestOrigin(r, adminOrigin) ||
				!adminauth.ValidateCSRF(value.csrfTokenHash, r.Header.Get("X-CSRF-Token")) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "csrf_invalid"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requiresCSRF(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return r.URL.Query().Get("reveal") == "true"
	}
	return true
}

func sameRequestOrigin(r *http.Request, expected string) bool {
	origin := strings.TrimSuffix(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin != "" {
		return origin == expected
	}
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return false
	}
	parsed, err := url.Parse(referer)
	if err != nil {
		return false
	}
	return parsed.Scheme+"://"+parsed.Host == expected
}
