package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	appsettings "github.com/djalmajr/tenancit/server/internal/settings"
)

type governanceFixture struct {
	server      *Server
	handler     http.Handler
	sessions    *adminauth.SessionManager
	current     adminauth.CreatedSession
	other       adminauth.CreatedSession
	cookieName  string
	adminOrigin string
}

func newGovernanceFixture(t *testing.T) governanceFixture {
	t.Helper()
	server, _ := newTestServer(t)
	authStore := adminauth.NewPostgresSessionStore(server.DB)
	sessions := adminauth.NewSessionManager(authStore, server.Cryptor, nil, time.Now, 8*time.Hour, 30*time.Minute)
	identity := adminauth.SessionIdentity{
		Issuer: "https://id.example.test", Subject: "security-admin-1", Label: "Grace Security",
		Roles: []adminauth.Role{adminauth.RoleSecurityAdmin}, Permissions: []string{"admin.read"},
	}
	current, err := sessions.Create(context.Background(), identity)
	if err != nil {
		t.Fatalf("create current session: %v", err)
	}
	other, err := sessions.Create(context.Background(), identity)
	if err != nil {
		t.Fatalf("create other session: %v", err)
	}
	const cookieName = "tenancit_session"
	const adminOrigin = "https://tenancit.example.test"
	server.ConfigureAdminAuth(adminauth.Config{
		Mode: adminauth.ModeOIDC, CookieName: cookieName, AdminOrigin: adminOrigin,
	}, nil, sessions)
	server.SetAdminAuthStore(authStore)
	return governanceFixture{
		server: server, handler: server.Routes(nil), sessions: sessions,
		current: current, other: other, cookieName: cookieName, adminOrigin: adminOrigin,
	}
}

func (f governanceFixture) request(t *testing.T, method, path string, body any, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := doJSONRequest(t, method, path, body)
	request := recorder.request
	request.AddCookie(&http.Cookie{Name: f.cookieName, Value: f.current.Token})
	if method != http.MethodGet {
		request.Header.Set("Origin", f.adminOrigin)
		request.Header.Set("X-CSRF-Token", f.current.CSRFToken)
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	f.handler.ServeHTTP(recorder.response, request)
	return recorder.response
}

type preparedRequest struct {
	request  *http.Request
	response *httptest.ResponseRecorder
}

func doJSONRequest(t *testing.T, method, path string, body any) preparedRequest {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	return preparedRequest{request: request, response: httptest.NewRecorder()}
}

func TestE2E_SettingsUseCASAndTransactionalAudit(t *testing.T) {
	fixture := newGovernanceFixture(t)
	initial := fixture.request(t, http.MethodGet, "/v1/admin/settings", nil, "")
	if initial.Code != http.StatusOK || initial.Header().Get("ETag") != `"settings-1"` {
		t.Fatalf("initial status=%d etag=%q body=%s", initial.Code, initial.Header().Get("ETag"), initial.Body)
	}

	withoutPrecondition := fixture.request(t, http.MethodPatch, "/v1/admin/settings", map[string]any{
		"values": map[string]string{appsettings.SessionIdleMinutes: "45"},
	}, "")
	if withoutPrecondition.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match status=%d", withoutPrecondition.Code)
	}

	updated := fixture.request(t, http.MethodPatch, "/v1/admin/settings", map[string]any{
		"values": map[string]string{appsettings.SessionIdleMinutes: "45", appsettings.DefaultLocale: "en-US"},
	}, `"settings-1"`)
	if updated.Code != http.StatusOK || updated.Header().Get("ETag") != `"settings-2"` {
		t.Fatalf("updated status=%d etag=%q body=%s", updated.Code, updated.Header().Get("ETag"), updated.Body)
	}
	stale := fixture.request(t, http.MethodPatch, "/v1/admin/settings", map[string]any{
		"values": map[string]string{appsettings.SessionIdleMinutes: "60"},
	}, `"settings-1"`)
	if stale.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale status=%d body=%s", stale.Code, stale.Body)
	}
	invalid := fixture.request(t, http.MethodPatch, "/v1/admin/settings", map[string]any{
		"values": map[string]string{"database_url": "postgres://must-not-be-stored"},
	}, `"settings-2"`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body)
	}

	var value string
	if err := fixture.server.DB.QueryRow(context.Background(), `SELECT value FROM admin_settings WHERE key = $1`, appsettings.SessionIdleMinutes).Scan(&value); err != nil || value != "45" {
		t.Fatalf("stored value=%q err=%v", value, err)
	}
	var auditCount int
	if err := fixture.server.DB.QueryRow(context.Background(), `SELECT count(*) FROM admin_audit_events WHERE action = 'settings.updated'`).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}

	if _, err := fixture.server.DB.Exec(context.Background(), `
		CREATE OR REPLACE FUNCTION fail_settings_audit() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action = 'settings.updated' THEN RAISE EXCEPTION 'audit unavailable'; END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER fail_settings_audit BEFORE INSERT ON admin_audit_events
		FOR EACH ROW EXECUTE FUNCTION fail_settings_audit()`); err != nil {
		t.Fatalf("install audit failure trigger: %v", err)
	}
	failedAudit := fixture.request(t, http.MethodPatch, "/v1/admin/settings", map[string]any{
		"values": map[string]string{appsettings.SessionIdleMinutes: "60"},
	}, `"settings-2"`)
	if failedAudit.Code != http.StatusInternalServerError {
		t.Fatalf("audit failure status=%d body=%s", failedAudit.Code, failedAudit.Body)
	}
	if _, err := fixture.server.DB.Exec(context.Background(), `DROP TRIGGER fail_settings_audit ON admin_audit_events; DROP FUNCTION fail_settings_audit()`); err != nil {
		t.Fatalf("remove audit failure trigger: %v", err)
	}
	if err := fixture.server.DB.QueryRow(context.Background(), `SELECT value FROM admin_settings WHERE key = $1`, appsettings.SessionIdleMinutes).Scan(&value); err != nil || value != "45" {
		t.Fatalf("audit failure did not roll back setting: value=%q err=%v", value, err)
	}
}

func TestE2E_SessionInventoryAndRevocationExcludeCurrent(t *testing.T) {
	fixture := newGovernanceFixture(t)
	listed := fixture.request(t, http.MethodGet, "/v1/admin/sessions", nil, "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body)
	}
	var sessions []adminSessionView
	mustJSON(t, listed, &sessions)
	if len(sessions) != 2 {
		t.Fatalf("sessions=%+v", sessions)
	}
	currentID := fixture.current.Identity.SessionID
	otherID := fixture.other.Identity.SessionID
	currentCount := 0
	for _, session := range sessions {
		if session.Current {
			currentCount++
			if session.ID != currentID {
				t.Fatalf("current session=%s want=%s", session.ID, currentID)
			}
		}
	}
	if currentCount != 1 {
		t.Fatalf("current count=%d", currentCount)
	}

	currentRevoke := fixture.request(t, http.MethodDelete, "/v1/admin/sessions/"+currentID, nil, "")
	if currentRevoke.Code != http.StatusConflict {
		t.Fatalf("current revoke status=%d body=%s", currentRevoke.Code, currentRevoke.Body)
	}
	otherRevoke := fixture.request(t, http.MethodDelete, "/v1/admin/sessions/"+otherID, nil, "")
	if otherRevoke.Code != http.StatusNoContent {
		t.Fatalf("other revoke status=%d body=%s", otherRevoke.Code, otherRevoke.Body)
	}
	if _, err := fixture.sessions.Authenticate(context.Background(), fixture.other.Token); err == nil {
		t.Fatal("revoked session still authenticates")
	}

	third, err := fixture.sessions.Create(context.Background(), adminauth.SessionIdentity{
		Issuer: "https://id.example.test", Subject: "security-admin-1", Label: "Grace Security",
		Roles: []adminauth.Role{adminauth.RoleSecurityAdmin}, Permissions: []string{"admin.read"},
	})
	if err != nil {
		t.Fatalf("create third session: %v", err)
	}
	principalRevoke := fixture.request(t, http.MethodPost, "/v1/admin/sessions/revoke-principal", map[string]string{
		"issuer": "https://id.example.test", "subject": "security-admin-1",
	}, "")
	if principalRevoke.Code != http.StatusOK {
		t.Fatalf("principal revoke status=%d body=%s", principalRevoke.Code, principalRevoke.Body)
	}
	if _, err := fixture.sessions.Authenticate(context.Background(), third.Token); err == nil {
		t.Fatal("principal revocation left another session active")
	}
	if _, err := fixture.sessions.Authenticate(context.Background(), fixture.current.Token); err != nil {
		t.Fatalf("principal revocation invalidated current session: %v", err)
	}
}
