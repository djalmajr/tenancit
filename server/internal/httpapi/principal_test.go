package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdminPermissionRejectsMissingPrincipal(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()

	requireAdminPermission(permissionAdminRead)(next).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil),
	)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", rec.Code)
	}
	if called {
		t.Fatal("handler was called without an authenticated principal")
	}
}

// Mutation captured: making hasPermission always allow reaches the handler and returns 204 instead of 403.
func TestRequireAdminPermissionRejectsMissingPermission(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/tenants/tenant-id", nil)
	req = req.WithContext(contextWithPrincipal(
		req.Context(),
		newPrincipal(principalKindSharedAdminToken, "primary", permissionAdminRead),
	))
	rec := httptest.NewRecorder()

	requireAdminPermission(permissionTenantHardDelete)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("code=%d want 403", rec.Code)
	}
	if called {
		t.Fatal("handler was called without the required permission")
	}
}

func TestRequireAdminPermissionAllowsMatchingPermission(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/tenants/tenant-id", nil)
	req = req.WithContext(contextWithPrincipal(
		req.Context(),
		newPrincipal(principalKindSharedAdminToken, "primary", permissionTenantHardDelete),
	))
	rec := httptest.NewRecorder()

	requireAdminPermission(permissionTenantHardDelete)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("code=%d want 204", rec.Code)
	}
	if !called {
		t.Fatal("handler was not called with the required permission")
	}
}

func TestRequireSecretRevealPermissionMatchesHandlerSemantics(t *testing.T) {
	tests := []struct {
		name        string
		permissions []adminPermission
		url         string
		wantCode    int
	}{
		{name: "masked list", url: "/v1/admin/tenants/id/resources", wantCode: http.StatusNoContent},
		{name: "explicit false", url: "/v1/admin/tenants/id/resources?reveal=false", wantCode: http.StatusNoContent},
		{name: "non-canonical true", url: "/v1/admin/tenants/id/resources?reveal=TRUE", wantCode: http.StatusNoContent},
		{name: "reveal without permission", url: "/v1/admin/tenants/id/resources?reveal=true", wantCode: http.StatusForbidden},
		{
			name:        "reveal with permission",
			permissions: []adminPermission{permissionSecretReveal},
			url:         "/v1/admin/tenants/id/resources?reveal=true",
			wantCode:    http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(contextWithPrincipal(
				req.Context(),
				newPrincipal(principalKindSharedAdminToken, "primary", tt.permissions...),
			))
			rec := httptest.NewRecorder()

			requireSecretRevealPermission(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("code=%d want %d", rec.Code, tt.wantCode)
			}
			if called != (tt.wantCode == http.StatusNoContent) {
				t.Fatalf("called=%v want %v", called, tt.wantCode == http.StatusNoContent)
			}
		})
	}
}

func TestOIDCPrincipalCarriesDurableIdentityAndSession(t *testing.T) {
	got, err := newOIDCPrincipal(
		"https://id.example.test",
		"user-123",
		"Ada Lovelace",
		"session-456",
		[]adminRole{roleViewer, roleOperator},
	)
	if err != nil {
		t.Fatalf("newOIDCPrincipal: %v", err)
	}
	if got.Kind != principalKindOIDCUser {
		t.Fatalf("kind=%q want %q", got.Kind, principalKindOIDCUser)
	}
	if got.Issuer != "https://id.example.test" || got.Subject != "user-123" {
		t.Fatalf("durable identity=%q/%q", got.Issuer, got.Subject)
	}
	if got.Label != "Ada Lovelace" || got.SessionID != "session-456" {
		t.Fatalf("display/session=%q/%q", got.Label, got.SessionID)
	}
	for _, permission := range []adminPermission{
		permissionAdminRead,
		permissionTenantWrite,
		permissionResourceWrite,
	} {
		if !got.hasPermission(permission) {
			t.Fatalf("OIDC principal missing permission %q", permission)
		}
	}
	if got.hasPermission(permissionSecretReveal) {
		t.Fatal("operator unexpectedly received secret.reveal")
	}
}

func TestOIDCPrincipalRejectsIncompleteOrUnknownIdentity(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		subject string
		roles   []adminRole
	}{
		{name: "missing issuer", subject: "user", roles: []adminRole{roleViewer}},
		{name: "missing subject", issuer: "https://id.example.test", roles: []adminRole{roleViewer}},
		{name: "missing role", issuer: "https://id.example.test", subject: "user"},
		{name: "unknown role", issuer: "https://id.example.test", subject: "user", roles: []adminRole{"owner"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := newOIDCPrincipal(tt.issuer, tt.subject, "User", "session", tt.roles); err == nil {
				t.Fatal("expected fail-closed principal validation")
			}
		})
	}
}

func TestBreakGlassPrincipalIsDistinctAndLeastPrivilege(t *testing.T) {
	got := newBreakGlassPrincipal("primary")
	if got.Kind != principalKindBreakGlass || got.Subject != "admin-token:primary" {
		t.Fatalf("principal=%+v", got)
	}
	if !got.hasPermission(permissionAdminRead) || !got.hasPermission(permissionAuditRead) {
		t.Fatal("break-glass principal cannot inspect recovery state")
	}
	if got.hasPermission(permissionTenantHardDelete) || got.hasPermission(permissionSecretReveal) {
		t.Fatal("break-glass principal received destructive or reveal permission")
	}
}
