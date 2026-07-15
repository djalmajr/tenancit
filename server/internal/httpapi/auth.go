package httpapi

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/telemetry"
)

// apiKeyLookup resolves a key hash to a client; error if absent.
type apiKeyLookup interface {
	GetAPIClientAuthByHash(ctx context.Context, keyHash string) (db.GetAPIClientAuthByHashRow, error)
}

type apiClientPrincipal struct {
	ID       string
	Name     string
	Scopes   map[string]struct{}
	RPMLimit *int32
}

type apiClientPrincipalContextKey struct{}

func contextWithAPIClientPrincipal(ctx context.Context, value apiClientPrincipal) context.Context {
	return context.WithValue(ctx, apiClientPrincipalContextKey{}, value)
}

func apiClientPrincipalFromContext(ctx context.Context) (apiClientPrincipal, bool) {
	value, ok := ctx.Value(apiClientPrincipalContextKey{}).(apiClientPrincipal)
	return value, ok
}

// RequireAPIKey authenticates consumer requests via Authorization: Bearer,
// matched by hash against api_clients (RN-09). Revoked/unknown -> 401.
func RequireAPIKey(q apiKeyLookup, now func() time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				telemetry.RecordSecurityDecision(r.Context(), "api_key_auth", "denied")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_api_key"})
				return
			}
			client, err := q.GetAPIClientAuthByHash(r.Context(), service.HashAPIKey(token))
			expired := !now().UTC().Before(client.ExpiresAt)
			if err != nil || client.Status != "active" || expired {
				telemetry.RecordSecurityDecision(r.Context(), "api_key_auth", "denied")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_api_key"})
				return
			}
			scopes := make(map[string]struct{}, len(client.Scopes))
			for _, scope := range client.Scopes {
				scopes[scope] = struct{}{}
			}
			principal := apiClientPrincipal{
				ID: client.ID.String(), Name: client.Name, Scopes: scopes, RPMLimit: &client.RpmLimit,
			}
			telemetry.RecordSecurityDecision(r.Context(), "api_key_auth", "success")
			next.ServeHTTP(w, r.WithContext(contextWithAPIClientPrincipal(r.Context(), principal)))
		})
	}
}

func RequireAPIClientScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := apiClientPrincipalFromContext(r.Context())
			if !ok {
				telemetry.RecordSecurityDecision(r.Context(), "api_scope", "denied")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_api_key"})
				return
			}
			if _, allowed := principal.Scopes[scope]; !allowed {
				telemetry.RecordSecurityDecision(r.Context(), "api_scope", "denied")
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient_scope"})
				return
			}
			telemetry.RecordSecurityDecision(r.Context(), "api_scope", "success")
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminToken protects the human/admin API surface with TENANCIT_ADMIN_TOKEN.
func RequireAdminToken(adminTokenHash string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			hash := service.HashAPIKey(token)
			if token == "" || subtle.ConstantTimeCompare([]byte(hash), []byte(adminTokenHash)) != 1 {
				telemetry.RecordSecurityDecision(r.Context(), "admin_identity", "denied")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin token"})
				return
			}
			value := newPrincipal(
				principalKindSharedAdminToken,
				"primary",
				sharedAdminPermissions[:]...,
			)
			telemetry.RecordSecurityDecision(r.Context(), "admin_identity", "success")
			next.ServeHTTP(w, r.WithContext(contextWithPrincipal(r.Context(), value)))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	scheme, token, ok := strings.Cut(h, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func constantTimeCredentialMatch(expectedHash, token string) bool {
	if expectedHash == "" || token == "" {
		return false
	}
	hash := service.HashAPIKey(token)
	return subtle.ConstantTimeCompare([]byte(hash), []byte(expectedHash)) == 1
}
