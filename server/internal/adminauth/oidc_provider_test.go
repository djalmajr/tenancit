package adminauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestClaimsFromMapUsesStableSubjectAndClosedRoleClaim(t *testing.T) {
	claims, err := claimsFromMap(map[string]any{
		"iss": "https://id.example.test", "sub": "subject-1", "nonce": "nonce-1",
		"name": "Ada", "groups": []any{"viewers", "operators"},
	}, "https://id.example.test", "groups")
	if err != nil {
		t.Fatalf("claimsFromMap: %v", err)
	}
	if claims.Subject != "subject-1" || claims.Label != "Ada" || len(claims.RoleValues) != 2 {
		t.Fatalf("claims=%+v", claims)
	}
}

func TestProviderRequestsGroupsScopeForGroupsRoleClaim(t *testing.T) {
	provider := &Provider{
		oauthConfig: oauth2.Config{
			ClientID: "tenancit",
			Endpoint: oauth2.Endpoint{AuthURL: "https://id.example.test/authorize"},
			Scopes:   []string{"openid", "profile", "email", "groups"},
		},
	}
	authorizationURL := provider.AuthorizationURL("state", "nonce", "challenge")
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	if !strings.Contains(parsed.Query().Get("scope"), "groups") {
		t.Fatalf("scope=%q", parsed.Query().Get("scope"))
	}
}

func TestClaimsFromMapRejectsMalformedRoleClaim(t *testing.T) {
	for _, value := range []any{nil, 42, []any{}, []any{"operators", 42}} {
		if _, err := claimsFromMap(map[string]any{
			"sub": "subject-1", "nonce": "nonce-1", "groups": value,
		}, "https://id.example.test", "groups"); err == nil {
			t.Fatalf("malformed role claim accepted: %#v", value)
		}
	}
}

func TestNewProviderFailsClosedWhenDiscoveryIsUnavailable(t *testing.T) {
	identityProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer identityProvider.Close()
	_, err := NewProvider(context.Background(), OIDCConfig{
		Issuer: identityProvider.URL, ClientID: "tenancit", ClientSecret: "secret", RedirectURL: "https://tenancit.example.test/v1/auth/callback",
	})
	if err == nil {
		t.Fatal("unavailable discovery endpoint was accepted")
	}
}
