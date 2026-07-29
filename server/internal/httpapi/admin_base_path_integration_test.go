package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
)

type prefixedOIDCProvider struct {
	nonce string
}

func (p *prefixedOIDCProvider) AuthorizationURL(state, nonce, _ string) string {
	p.nonce = nonce
	query := url.Values{"state": []string{state}}
	return "https://id.example.test/authorize?" + query.Encode()
}

func (p *prefixedOIDCProvider) Exchange(context.Context, string, string) (adminauth.OIDCClaims, error) {
	return adminauth.OIDCClaims{
		Issuer:     "https://id.example.test",
		Subject:    "human-admin",
		Label:      "Human Admin",
		Nonce:      p.nonce,
		RoleValues: []string{"admins"},
	}, nil
}

func TestE2E_PrefixedOIDCUsesScopedCookieAcrossHumanAdminFlow(t *testing.T) {
	// Mutation captured by the initial RED: hard-coding cookie Path=/ leaks
	// the Tenancit session to sibling applications on the shared admin host.
	server, _ := newTestServer(t)
	store := adminauth.NewPostgresSessionStore(server.DB)
	sessions := adminauth.NewSessionManager(store, server.Cryptor, nil, time.Now, 8*time.Hour, 30*time.Minute)
	provider := &prefixedOIDCProvider{}
	manager := adminauth.NewOIDCManager(adminauth.OIDCConfig{
		BasePath:     "/tenancit",
		Issuer:       "https://id.example.test",
		RoleMappings: map[string]adminauth.Role{"admins": adminauth.RoleSecurityAdmin},
	}, provider, store, sessions, server.Cryptor, nil, time.Now)
	server.SetAdminAuthStore(store)
	server.ConfigureAdminAuth(adminauth.Config{
		AdminOrigin:  "https://admin.example.test",
		BasePath:     "/tenancit",
		CookieName:   "__Secure-tenancit_session",
		CookiePath:   "/tenancit",
		CookieSecure: true,
		Mode:         adminauth.ModeOIDC,
	}, manager, sessions)
	handler := server.Routes(nil)

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(
		http.MethodGet,
		"/tenancit/v1/auth/login?return_to=%2Ftenancit%2Ftenants",
		nil,
	))
	if login.Code != http.StatusFound {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body)
	}
	authorizationURL, err := url.Parse(login.Header().Get("Location"))
	if err != nil || authorizationURL.Query().Get("state") == "" {
		t.Fatalf("authorization location=%q err=%v", login.Header().Get("Location"), err)
	}

	callback := httptest.NewRecorder()
	callbackURL := "/tenancit/v1/auth/callback?state=" +
		url.QueryEscape(authorizationURL.Query().Get("state")) + "&code=authorization-code"
	handler.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, callbackURL, nil))
	if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != "/tenancit/tenants" {
		t.Fatalf("callback=%d location=%q body=%s", callback.Code, callback.Header().Get("Location"), callback.Body)
	}
	sessionCookie := findCookie(t, callback.Result().Cookies(), "__Secure-tenancit_session")
	if sessionCookie.Path != "/tenancit" || !sessionCookie.Secure || !sessionCookie.HttpOnly ||
		sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie=%+v", sessionCookie)
	}

	overviewRequest := httptest.NewRequest(http.MethodGet, "/tenancit/v1/admin/overview", nil)
	overviewRequest.AddCookie(sessionCookie)
	overview := httptest.NewRecorder()
	handler.ServeHTTP(overview, overviewRequest)
	if overview.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", overview.Code, overview.Body)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/tenancit/v1/auth/session", nil)
	sessionRequest.AddCookie(sessionCookie)
	session := httptest.NewRecorder()
	handler.ServeHTTP(session, sessionRequest)
	if session.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", session.Code, session.Body)
	}
	var sessionBody struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.NewDecoder(session.Body).Decode(&sessionBody); err != nil || sessionBody.CSRFToken == "" {
		t.Fatalf("session body=%s err=%v", session.Body, err)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/tenancit/v1/auth/logout", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.Header.Set("Origin", "https://admin.example.test")
	logoutRequest.Header.Set("X-CSRF-Token", sessionBody.CSRFToken)
	logout := httptest.NewRecorder()
	handler.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body)
	}
	clearedCookie := findCookie(t, logout.Result().Cookies(), "__Secure-tenancit_session")
	if clearedCookie.Path != "/tenancit" || clearedCookie.MaxAge != -1 {
		t.Fatalf("cleared cookie=%+v", clearedCookie)
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %+v", name, cookies)
	return nil
}
