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
