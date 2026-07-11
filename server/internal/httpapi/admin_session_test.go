package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/service"
)

type fakeSessionAuthenticator struct {
	identity adminauth.SessionIdentity
	err      error
	token    string
}

func TestRequireAdminIdentityAcceptsOnlyExplicitBreakGlassBearer(t *testing.T) {
	token := "high-entropy-break-glass-token-value"
	tests := []struct {
		name       string
		header     string
		hash       string
		wantStatus int
		wantKind   principalKind
	}{
		{name: "disabled", header: "Bearer " + token, wantStatus: http.StatusUnauthorized},
		{name: "wrong token", header: "Bearer wrong", hash: service.HashAPIKey(token), wantStatus: http.StatusUnauthorized},
		{name: "enabled", header: "Bearer " + token, hash: service.HashAPIKey(token), wantStatus: http.StatusNoContent, wantKind: principalKindBreakGlass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil)
			req.Header.Set("Authorization", tt.header)
			recorder := httptest.NewRecorder()
			var got principal
			RequireAdminIdentity(nil, "session", tt.hash, "rotation-1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, _ = principalFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(recorder, req)
			if recorder.Code != tt.wantStatus || got.Kind != tt.wantKind {
				t.Fatalf("status=%d principal=%+v", recorder.Code, got)
			}
		})
	}
}

func TestPrincipalCaptureObservesDownstreamOIDCIdentity(t *testing.T) {
	ctx, capture := contextWithPrincipalCapture(context.Background())
	value, err := newOIDCPrincipal("https://id.example.test", "user-1", "Ada", "session-1", []adminRole{roleViewer})
	if err != nil {
		t.Fatalf("newOIDCPrincipal: %v", err)
	}
	_ = contextWithPrincipal(ctx, value)
	if !capture.ok || capture.value.Issuer != "https://id.example.test" || capture.value.Subject != "user-1" {
		t.Fatalf("capture=%+v", capture)
	}
}

func TestRequireAdminCSRFIgnoresCookieDefenseForExplicitBreakGlass(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/recovery", nil)
	req = req.WithContext(contextWithPrincipal(req.Context(), newBreakGlassPrincipal("rotation-1")))
	recorder := httptest.NewRecorder()
	RequireAdminCSRF("https://tenancit.example.test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func (f *fakeSessionAuthenticator) Authenticate(_ context.Context, token string) (adminauth.SessionIdentity, error) {
	f.token = token
	return f.identity, f.err
}

func TestRequireAdminSessionAuthenticatesOpaqueCookie(t *testing.T) {
	authenticator := &fakeSessionAuthenticator{identity: adminauth.SessionIdentity{
		Issuer: "https://id.example.test", Subject: "user-1", Label: "Ada", SessionID: "session-1",
		Roles: []adminauth.Role{adminauth.RoleOperator}, Permissions: []string{"admin.read"},
		CSRFTokenHash: adminauth.HashCredential("csrf-token"),
	}}
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil)
	req.AddCookie(&http.Cookie{Name: "__Host-tenancit_session", Value: "opaque-session"})
	rec := httptest.NewRecorder()
	var got principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = principalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	RequireAdminSession(authenticator, "__Host-tenancit_session")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent || authenticator.token != "opaque-session" {
		t.Fatalf("code=%d token=%q", rec.Code, authenticator.token)
	}
	if got.Kind != principalKindOIDCUser || got.Issuer != authenticator.identity.Issuer || got.SessionID != "session-1" {
		t.Fatalf("principal=%+v", got)
	}
}

func TestRequireAdminCSRFProtectsUnsafeMethodsAndReveal(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		origin     string
		csrf       string
		wantStatus int
	}{
		{name: "safe get", method: http.MethodGet, url: "/v1/admin/overview", wantStatus: http.StatusNoContent},
		{name: "reveal get requires csrf", method: http.MethodGet, url: "/v1/admin/resources?reveal=true", wantStatus: http.StatusForbidden},
		{name: "post without origin", method: http.MethodPost, url: "/v1/admin/tenants", csrf: "csrf-token", wantStatus: http.StatusForbidden},
		{name: "post wrong origin", method: http.MethodPost, url: "/v1/admin/tenants", origin: "https://evil.test", csrf: "csrf-token", wantStatus: http.StatusForbidden},
		{name: "post wrong csrf", method: http.MethodPost, url: "/v1/admin/tenants", origin: "https://tenancit.example.test", csrf: "wrong", wantStatus: http.StatusForbidden},
		{name: "post valid", method: http.MethodPost, url: "/v1/admin/tenants", origin: "https://tenancit.example.test", csrf: "csrf-token", wantStatus: http.StatusNoContent},
		{name: "reveal valid", method: http.MethodGet, url: "/v1/admin/resources?reveal=true", origin: "https://tenancit.example.test", csrf: "csrf-token", wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.csrf != "" {
				req.Header.Set("X-CSRF-Token", tt.csrf)
			}
			value := newPrincipal(principalKindOIDCUser, "user-1", permissionAdminRead)
			value.csrfTokenHash = adminauth.HashCredential("csrf-token")
			req = req.WithContext(contextWithPrincipal(req.Context(), value))
			rec := httptest.NewRecorder()

			RequireAdminCSRF("https://tenancit.example.test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("code=%d want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
