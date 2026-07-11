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
	server.ConfigureAdminAuth(adminauth.Config{Mode: adminauth.ModeOIDC}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/auth/config", nil)

	server.Routes(nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body struct {
		Mode     adminauth.Mode `json:"mode"`
		LoginURL string         `json:"login_url"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Mode != adminauth.ModeOIDC || body.LoginURL != "/v1/auth/login" {
		t.Fatalf("response = %+v", body)
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
