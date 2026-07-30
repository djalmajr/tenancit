package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
)

func TestAdminAuthConfigIsPublicAndDescribesOIDCLogin(t *testing.T) {
	server := &Server{}
	server.ConfigureAdminAuth(adminauth.Config{Mode: adminauth.ModeOIDC, BasePath: "/tenancit"}, nil, nil)
	handler := server.Routes(nil)

	// Mutation captured by the initial RED: only registering the root routes
	// left human administration unreachable below the shared-host prefix.
	for _, requestPath := range []string{"/v1/auth/config", "/tenancit/v1/auth/config"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, requestPath, nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", requestPath, recorder.Code, http.StatusOK)
		}
		var body struct {
			Mode     adminauth.Mode `json:"mode"`
			LoginURL string         `json:"login_url"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Mode != adminauth.ModeOIDC || body.LoginURL != "/tenancit/v1/auth/login" {
			t.Fatalf("%s response = %+v", requestPath, body)
		}
	}
}

func TestPrefixedRoutesExposeOnlyHumanAdministrationAliases(t *testing.T) {
	server := &Server{}
	server.ConfigureAdminAuth(adminauth.Config{Mode: adminauth.ModeOIDC, BasePath: "/tenancit"}, nil, nil)
	handler := server.Routes(nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "root login remains", path: "/v1/auth/login", wantStatus: http.StatusServiceUnavailable},
		{name: "prefixed login alias", path: "/tenancit/v1/auth/login", wantStatus: http.StatusServiceUnavailable},
		{name: "prefixed admin alias", path: "/tenancit/v1/admin/overview", wantStatus: http.StatusUnauthorized},
		{name: "consumer API is not duplicated", path: "/tenancit/v1/identify", wantStatus: http.StatusNotFound},
		{name: "prefix boundary is exact", path: "/tenancit-other/v1/auth/config", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("%s status = %d, want %d; body=%s", tt.path, recorder.Code, tt.wantStatus, recorder.Body)
			}
		})
	}
}

func TestAdminAuthConfigOmitsLoginForLegacyMode(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/auth/config", nil)

	server.Routes(nil).ServeHTTP(recorder, request)

	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["mode"] != string(adminauth.ModeLegacySharedToken) {
		t.Fatalf("mode = %v", body["mode"])
	}
	if _, exists := body["login_url"]; exists {
		t.Fatal("legacy response must not advertise an OIDC login URL")
	}
}

func TestAdminSessionResponseIsNeverStored(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)

	server.getAdminSession(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control = %q, want private, no-store", got)
	}
}
