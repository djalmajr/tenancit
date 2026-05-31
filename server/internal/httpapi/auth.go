package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/centralit/resource-tenant/server/internal/service"
	"github.com/centralit/resource-tenant/server/internal/store/db"
)

// apiKeyLookup resolves a key hash to a client; error if absent.
type apiKeyLookup interface {
	GetAPIClientByHash(ctx context.Context, keyHash string) (db.ApiClient, error)
}

// RequireAPIKey authenticates consumer requests via Authorization: Bearer,
// matched by hash against api_clients (RN-09). Revoked/unknown -> 401.
func RequireAPIKey(q apiKeyLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing api key"})
				return
			}
			client, err := q.GetAPIClientByHash(r.Context(), service.HashAPIKey(token))
			if err != nil || client.Status != "active" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return strings.TrimSpace(h)
}
