package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func TestE2E_AuditExportIsGovernedAndOneShot(t *testing.T) {
	fixture := newGovernanceFixture(t)
	now := time.Now().UTC()
	if _, err := fixture.server.DB.Exec(t.Context(), `INSERT INTO admin_audit_events
		(occurred_at,request_id,actor_kind,actor_subject,action,target_type,target_id,result,http_method,route_template,http_status,metadata)
		VALUES ($1,'export-source','shared_admin_token','primary','tenant.created','tenant','export-tenant','success','POST','/v1/admin/tenants',201,'{}')`, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	prepared := doJSONRequest(t, http.MethodPost, "/v1/admin/audit-exports", map[string]any{
		"format": "csv", "filters": map[string]any{"from": now.Add(-24 * time.Hour), "to": now},
	})
	prepared.request.AddCookie(&http.Cookie{Name: fixture.cookieName, Value: fixture.current.Token})
	prepared.request.Header.Set("Origin", fixture.adminOrigin)
	prepared.request.Header.Set("X-CSRF-Token", fixture.current.CSRFToken)
	prepared.request.Header.Set("Idempotency-Key", uuid.NewString())
	fixture.handler.ServeHTTP(prepared.response, prepared.request)
	if prepared.response.Code != http.StatusCreated {
		t.Fatalf("create export status=%d body=%s", prepared.response.Code, prepared.response.Body)
	}
	var created struct {
		ID     uuid.UUID `json:"id"`
		Status string    `json:"status"`
	}
	if err := json.Unmarshal(prepared.response.Body.Bytes(), &created); err != nil || created.Status != "ready" {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	download := fixture.request(t, http.MethodGet, "/v1/admin/audit-exports/"+created.ID.String()+"/download", nil, "")
	if download.Code != http.StatusOK || download.Header().Get("Cache-Control") != "no-store" || download.Header().Get("Content-Disposition") == "" {
		t.Fatalf("download status=%d headers=%v body=%s", download.Code, download.Header(), download.Body)
	}
	second := fixture.request(t, http.MethodGet, "/v1/admin/audit-exports/"+created.ID.String()+"/download", nil, "")
	if second.Code != http.StatusGone {
		t.Fatalf("second download status=%d body=%s", second.Code, second.Body)
	}
}

func TestE2E_AuthenticationLifecycleIsAudited(t *testing.T) {
	fixture := newGovernanceFixture(t)
	request := doJSONRequest(t, http.MethodGet, "/v1/auth/callback", nil).request
	identity := adminauth.SessionIdentity{Issuer: "https://id.example.test", Subject: "auditor-login", Label: "Auditor", SessionID: uuid.NewString()}
	var auditErr error
	middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		auditErr = fixture.server.insertAuthenticationAudit(r, identity, "admin.login_succeeded", "success", http.StatusSeeOther, "")
	})).ServeHTTP(httptest.NewRecorder(), request)
	if auditErr != nil {
		t.Fatalf("audit login: %v", auditErr)
	}
	logout := fixture.request(t, http.MethodPost, "/v1/auth/logout", nil, "")
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body)
	}
	var login, logoutCount int
	if err := fixture.server.DB.QueryRow(t.Context(), `SELECT count(*) FROM admin_audit_events WHERE action='admin.login_succeeded' AND actor_issuer='https://id.example.test' AND actor_subject='auditor-login'`).Scan(&login); err != nil {
		t.Fatal(err)
	}
	if err := fixture.server.DB.QueryRow(t.Context(), `SELECT count(*) FROM admin_audit_events WHERE action='admin.logout_succeeded' AND actor_subject='security-admin-1'`).Scan(&logoutCount); err != nil {
		t.Fatal(err)
	}
	if login != 1 || logoutCount != 1 {
		t.Fatalf("login=%d logout=%d", login, logoutCount)
	}
}

func TestE2E_LegalHoldLifecycleIsTransactionalAndAudited(t *testing.T) {
	fixture := newGovernanceFixture(t)
	now := time.Now().UTC()
	created := fixture.request(t, http.MethodPost, "/v1/admin/audit-legal-holds", map[string]any{
		"from": now.AddDate(0, -1, 0), "to": now, "reason": "incident-42",
	}, "")
	if created.Code != http.StatusCreated {
		t.Fatalf("create hold status=%d body=%s", created.Code, created.Body)
	}
	var body struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	released := fixture.request(t, http.MethodPost, "/v1/admin/audit-legal-holds/"+body.ID.String()+"/release", nil, "")
	if released.Code != http.StatusNoContent {
		t.Fatalf("release status=%d body=%s", released.Code, released.Body)
	}
	var count int
	if err := fixture.server.DB.QueryRow(t.Context(), `SELECT count(*) FROM admin_audit_events WHERE target_id=$1 AND action IN ('audit.legal_hold_created','audit.legal_hold_released')`, body.ID.String()).Scan(&count); err != nil || count != 2 {
		t.Fatalf("audit count=%d err=%v", count, err)
	}
}
