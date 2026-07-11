package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/service"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeLookup struct {
	clients map[string]db.GetAPIClientAuthByHashRow
}

func (f fakeLookup) GetAPIClientAuthByHash(_ context.Context, h string) (db.GetAPIClientAuthByHashRow, error) {
	c, ok := f.clients[h]
	if !ok {
		return db.GetAPIClientAuthByHashRow{}, errors.New("not found")
	}
	return c, nil
}

func protected(lk apiKeyLookup) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	return RequireAPIKey(lk, time.Now)(ok)
}

func TestAuth_NoToken_401(t *testing.T) {
	rec := httptest.NewRecorder()
	protected(fakeLookup{}).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", rec.Code)
	}
}

func TestAuth_ValidToken_200(t *testing.T) {
	tok := "tnc_abc"
	lk := fakeLookup{clients: map[string]db.GetAPIClientAuthByHashRow{service.HashAPIKey(tok): {
		Status: "active", Scopes: []string{service.ScopeTenantIdentify},
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	protected(lk).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d want 200", rec.Code)
	}
}

func TestAuth_RawTokenWithoutBearer_401(t *testing.T) {
	tok := "tnc_abc"
	lk := fakeLookup{clients: map[string]db.GetAPIClientAuthByHashRow{service.HashAPIKey(tok): {Status: "active"}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", tok)
	protected(lk).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 for non-Bearer authorization", rec.Code)
	}
}

func TestAuth_RevokedToken_401(t *testing.T) {
	tok := "tnc_revoked"
	lk := fakeLookup{clients: map[string]db.GetAPIClientAuthByHashRow{service.HashAPIKey(tok): {Status: "revoked"}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	protected(lk).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 for revoked", rec.Code)
	}
}

func TestAuth_ExpiredToken_401AtBoundary(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	tok := "tnc_expired"
	lk := fakeLookup{clients: map[string]db.GetAPIClientAuthByHashRow{service.HashAPIKey(tok): {
		Status: "active", Scopes: []string{service.ScopeTenantIdentify},
		ExpiresAt: pgtype.Timestamptz{Time: now, Valid: true},
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	RequireAPIKey(lk, func() time.Time { return now })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 for expired", rec.Code)
	}
}

func TestAuth_MissingScope_403(t *testing.T) {
	ctx := contextWithAPIClientPrincipal(context.Background(), apiClientPrincipal{
		Scopes: map[string]struct{}{service.ScopeTenantIdentify: {}},
	})
	req := httptest.NewRequest("GET", "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	RequireAPIClientScope(service.ScopeResourceResolve)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code=%d want 403", rec.Code)
	}
}

func TestAdminAuthAddsNonSecretSharedPrincipal(t *testing.T) {
	token := "admin-secret-token"
	tokenHash := service.HashAPIKey(token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	var got principal
	found := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, found = principalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	RequireAdminToken(tokenHash)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("code=%d want 204", rec.Code)
	}
	if !found {
		t.Fatal("authenticated admin principal missing from request context")
	}
	if got.Kind != principalKindSharedAdminToken || got.Subject != "primary" {
		t.Fatalf("principal=%+v want shared_admin_token/primary", got)
	}
	if got.Subject == token || got.Subject == tokenHash {
		t.Fatal("principal subject contains reusable token material")
	}
	for _, permission := range sharedAdminPermissions {
		if !got.hasPermission(permission) {
			t.Fatalf("shared admin principal missing permission %q", permission)
		}
	}
}
