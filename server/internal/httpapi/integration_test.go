package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/adminauth"
	"github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/djalmajr/tenancit/server/internal/testsupport"
	"github.com/google/uuid"
)

const testAdminToken = "admin-test-token"

// newTestServer builds a Server backed by a real ephemeral Postgres
// (testcontainers) and a deterministic cryptor.
func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	pool := testsupport.NewDB(t)
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 7)
	}
	c, _ := crypto.New(map[int][]byte{1: k}, 1)
	srv := NewServer(pool, c, testAdminToken)
	return srv, srv.Routes(nil)
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, &buf)
	if strings.HasPrefix(path, "/v1/admin") {
		req.Header.Set("Authorization", "Bearer "+testAdminToken)
		if method == http.MethodPost {
			req.Header.Set("Idempotency-Key", uuid.NewString())
		}
	}
	h.ServeHTTP(rec, req)
	return rec
}

func mustJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode %T: %v (status=%d body=%s)", v, err, rec.Code, rec.Body)
	}
}

// idOf decodes the "id" field from a JSON object response.
func idOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var m map[string]any
	mustJSON(t, rec, &m)
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatalf("no id in response: %s", rec.Body)
	}
	return id
}

// seedDefinition creates a postgres-like definition with host(req) + password(req,secret).
func seedDefinition(t *testing.T, h http.Handler, key string) string {
	t.Helper()
	rec := do(t, h, "POST", "/v1/admin/resource-definitions", map[string]string{"key": key, "name": "PG"})
	if rec.Code != 201 {
		t.Fatalf("create def: %d %s", rec.Code, rec.Body)
	}
	defID := idOf(t, rec)
	for _, f := range []map[string]any{
		{"key": "host", "required": true},
		{"key": "password", "required": true, "isSecret": true},
	} {
		if r := do(t, h, "POST", "/v1/admin/resource-definitions/"+defID+"/fields", f); r.Code != 201 {
			t.Fatalf("add field %v: %d %s", f, r.Code, r.Body)
		}
	}
	return defID
}

func seedTenant(t *testing.T, h http.Handler, slug, hostname string) string {
	t.Helper()
	rec := do(t, h, "POST", "/v1/admin/tenants", map[string]string{"slug": slug, "name": slug})
	if rec.Code != 201 {
		t.Fatalf("create tenant: %d %s", rec.Code, rec.Body)
	}
	id := idOf(t, rec)
	if hostname != "" {
		if r := do(t, h, "POST", "/v1/admin/tenants/"+id+"/domains", map[string]string{"hostname": hostname}); r.Code != 201 {
			t.Fatalf("add domain: %d %s", r.Code, r.Body)
		}
	}
	return id
}

func mintToken(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := do(t, h, "POST", "/v1/admin/api-clients", apiClientCreateBody("c"))
	if rec.Code != 201 {
		t.Fatalf("api client: %d %s", rec.Code, rec.Body)
	}
	var out struct {
		Token string `json:"token"`
	}
	mustJSON(t, rec, &out)
	return out.Token
}

func apiClientCreateBody(name string) map[string]any {
	return map[string]any{
		"name": name, "scopes": []string{"tenant:identify", "resource:resolve"},
		"rpm_limit": 300, "expires_at": time.Now().UTC().Add(90 * 24 * time.Hour),
	}
}

// Full admin→consumer golden path; secrets decrypt only at /v1/resolve.
func TestE2E_AdminCreateThenResolve(t *testing.T) {
	_, h := newTestServer(t)

	defID := seedDefinition(t, h, "pg-e2e")
	_ = defID
	tid := seedTenant(t, h, "e2e", "e2e.example.com")

	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", map[string]any{
		"definitionKey": "pg-e2e",
		"values":        map[string]string{"host": "db.e2e.internal", "password": "s3cr3t"},
	})
	if rec.Code != 201 {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}

	token := mintToken(t, h)
	req := httptest.NewRequest("GET", "/v1/resolve?hostname=e2e.example.com", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rrec := httptest.NewRecorder()
	h.ServeHTTP(rrec, req)
	if rrec.Code != 200 {
		t.Fatalf("resolve: %d %s", rrec.Code, rrec.Body)
	}
	var resolved struct {
		TenantSlug string `json:"tenantSlug"`
		Resources  []struct {
			DefinitionKey string            `json:"definitionKey"`
			Values        map[string]string `json:"values"`
		} `json:"resources"`
	}
	mustJSON(t, rrec, &resolved)
	if resolved.TenantSlug != "e2e" || len(resolved.Resources) != 1 {
		t.Fatalf("bad resolve: %+v", resolved)
	}
	// Mutation captured: if the resolver skipped decryption (RN-05), password
	// would come back as ciphertext/garbage instead of the cleartext.
	if got := resolved.Resources[0].Values["password"]; got != "s3cr3t" {
		t.Fatalf("password not decrypted: %q", got)
	}

	oneReq := httptest.NewRequest("GET", "/v1/resolve/e2e.example.com/resources/pg-e2e", nil)
	oneReq.Header.Set("Authorization", "Bearer "+token)
	oneRec := httptest.NewRecorder()
	h.ServeHTTP(oneRec, oneReq)
	if oneRec.Code != http.StatusOK {
		t.Fatalf("resolve one: %d %s", oneRec.Code, oneRec.Body)
	}
	if got := oneRec.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("resolve-one Cache-Control = %q, want private, no-store", got)
	}

	// Resolve without an API key must be rejected (RN-09).
	// Mutation captured: removing RequireAPIKey middleware would make this 200.
	rrec = httptest.NewRecorder()
	h.ServeHTTP(rrec, httptest.NewRequest("GET", "/v1/resolve?hostname=e2e.example.com", nil))
	if rrec.Code != 401 {
		t.Fatalf("expected 401 without key, got %d", rrec.Code)
	}
}

// Resolve by tenantId (the x-tenant-id identity path used by apps), plus the
// ETag / conditional-GET contract: a matching If-None-Match returns 304 without
// a body, and any resource mutation flips the tag.
func TestE2E_ResolveByTenantIdAndETag(t *testing.T) {
	_, h := newTestServer(t)

	seedDefinition(t, h, "pg-etag")
	tid := seedTenant(t, h, "acme", "acme.example.com")
	if r := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", map[string]any{
		"definitionKey": "pg-etag",
		"values":        map[string]string{"host": "db.acme.internal", "password": "p@ss"},
	}); r.Code != 201 {
		t.Fatalf("create resource: %d %s", r.Code, r.Body)
	}
	token := mintToken(t, h)

	resolve := func(query, ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/resolve?"+query, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// resolve by tenantId (slug) returns the tenant + a decrypted resource + ETag.
	bySlug := resolve("tenantId=acme", "")
	if bySlug.Code != 200 {
		t.Fatalf("resolve by tenantId: %d %s", bySlug.Code, bySlug.Body)
	}
	var got struct {
		TenantSlug string `json:"tenantSlug"`
		Resources  []struct {
			Values map[string]string `json:"values"`
		} `json:"resources"`
	}
	mustJSON(t, bySlug, &got)
	if got.TenantSlug != "acme" || len(got.Resources) != 1 || got.Resources[0].Values["password"] != "p@ss" {
		t.Fatalf("bad resolve by tenantId: %+v", got)
	}
	etag := bySlug.Header().Get("ETag")
	if etag == "" || bySlug.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("missing ETag/Cache-Control: etag=%q cc=%q", etag, bySlug.Header().Get("Cache-Control"))
	}

	// hostname and tenantId resolve the SAME tenant -> identical ETag.
	byHost := resolve("hostname=acme.example.com", "")
	if byHost.Code != 200 || byHost.Header().Get("ETag") != etag {
		t.Fatalf("hostname ETag %q != tenantId ETag %q (code %d)", byHost.Header().Get("ETag"), etag, byHost.Code)
	}

	// conditional GET with the current ETag -> 304, empty body (no re-decrypt).
	notMod := resolve("tenantId=acme", etag)
	if notMod.Code != http.StatusNotModified {
		t.Fatalf("expected 304 with matching If-None-Match, got %d %s", notMod.Code, notMod.Body)
	}
	if notMod.Body.Len() != 0 {
		t.Fatalf("304 should have empty body, got %q", notMod.Body)
	}
	if got := notMod.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("304 Cache-Control = %q, want private, no-store", got)
	}

	// mutating a resource flips the ETag -> stale conditional GET returns 200 fresh.
	var resources []struct {
		ID string `json:"id"`
	}
	mustJSON(t, do(t, h, "GET", "/v1/admin/tenants/"+tid+"/resources", nil), &resources)
	if len(resources) == 0 {
		t.Fatal("no resources to mutate")
	}
	if r := do(t, h, "PUT", "/v1/admin/tenants/"+tid+"/resources/"+resources[0].ID+"/status",
		map[string]string{"status": "inactive"}); r.Code != 200 {
		t.Fatalf("set resource status: %d %s", r.Code, r.Body)
	}
	stale := resolve("tenantId=acme", etag)
	if stale.Code != 200 {
		t.Fatalf("expected 200 after mutation (etag changed), got %d %s", stale.Code, stale.Body)
	}
	if stale.Header().Get("ETag") == etag {
		t.Fatal("ETag did not change after resource mutation")
	}

	// unknown tenantId -> 404.
	if r := resolve("tenantId=does-not-exist", ""); r.Code != 404 {
		t.Fatalf("expected 404 for unknown tenantId, got %d", r.Code)
	}

	// Ambiguous selectors are rejected instead of silently preferring one.
	if r := resolve("tenantId=acme&hostname=acme.example.com", ""); r.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for both tenantId and hostname, got %d: %s", r.Code, r.Body.String())
	}
}

// Suspending a tenant must make every consumer lookup indistinguishable from
// an unknown tenant, including paths that would otherwise return cleartext.
func TestE2E_InactiveTenant_NotResolvable(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg-inactive")
	tenantID := seedTenant(t, h, "inactive", "inactive.example.com")
	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-inactive",
		"values":        map[string]string{"host": "db.internal", "password": "must-not-leak"},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}
	token := mintToken(t, h)
	if rec := do(t, h, "PUT", "/v1/admin/tenants/"+tenantID, map[string]string{
		"name": "inactive", "slug": "inactive", "status": "inactive",
	}); rec.Code != http.StatusOK {
		t.Fatalf("deactivate tenant: %d %s", rec.Code, rec.Body)
	}

	paths := []string{
		"/v1/identify?hostname=inactive.example.com",
		"/v1/resolve?hostname=inactive.example.com",
		"/v1/resolve?tenantId=inactive",
		"/v1/resolve/inactive.example.com/resources/pg-inactive",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (%s)", path, rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "must-not-leak") {
			t.Errorf("GET %s leaked a secret: %s", path, rec.Body)
		}
	}
}

// /v1/identify maps a hostname to the tenant slug for the edge injector, with an
// ETag, and must never leak resources/secrets.
func TestE2E_Identify(t *testing.T) {
	_, h := newTestServer(t)
	tenantA := seedTenant(t, h, "acme", "acme.example.com")
	tenantB := seedTenant(t, h, "globex", "")
	token := mintToken(t, h)

	get := func(query, inm string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/identify?"+query, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if inm != "" {
			req.Header.Set("If-None-Match", inm)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	ok := get("hostname=acme.example.com", "")
	if ok.Code != 200 {
		t.Fatalf("identify: %d %s", ok.Code, ok.Body)
	}
	var id struct {
		TenantSlug string `json:"tenantSlug"`
	}
	mustJSON(t, ok, &id)
	if id.TenantSlug != "acme" {
		t.Fatalf("bad slug: %+v", id)
	}
	if strings.Contains(ok.Body.String(), "resources") || strings.Contains(ok.Body.String(), "values") {
		t.Fatalf("identify must not leak resources/secrets: %s", ok.Body)
	}
	etag := ok.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag on identify")
	}
	if got := ok.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("identify Cache-Control = %q, want private, no-cache", got)
	}
	if nm := get("hostname=acme.example.com", etag); nm.Code != http.StatusNotModified {
		t.Fatalf("expected 304 with matching If-None-Match, got %d", nm.Code)
	} else if got := nm.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("identify 304 Cache-Control = %q, want private, no-cache", got)
	}

	var domains []struct {
		ID string `json:"id"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantA+"/domains", nil), &domains)
	if len(domains) != 1 {
		t.Fatalf("tenant A domains = %d, want 1", len(domains))
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantA+"/domains/"+domains[0].ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("remove tenant A domain: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantB+"/domains", map[string]string{"hostname": "acme.example.com"}); rec.Code != http.StatusCreated {
		t.Fatalf("assign domain to tenant B: %d %s", rec.Code, rec.Body)
	}
	reassigned := get("hostname=acme.example.com", etag)
	if reassigned.Code != http.StatusOK {
		t.Fatalf("identify reassigned domain with stale ETag: %d %s", reassigned.Code, reassigned.Body)
	}
	var reassignedID identifyResponse
	mustJSON(t, reassigned, &reassignedID)
	if reassignedID.TenantSlug != "globex" {
		t.Fatalf("reassigned domain resolved to %q, want globex", reassignedID.TenantSlug)
	}
	if reassigned.Header().Get("ETag") == etag {
		t.Fatal("reassigned domain retained tenant A ETag")
	}
	if r := get("hostname=nope.example.com", ""); r.Code != 404 {
		t.Fatalf("expected 404 for unknown host, got %d", r.Code)
	}
	if r := get("", ""); r.Code != 400 {
		t.Fatalf("expected 400 for missing hostname, got %d", r.Code)
	}
	noauth := httptest.NewRecorder()
	h.ServeHTTP(noauth, httptest.NewRequest("GET", "/v1/identify?hostname=acme.example.com", nil))
	if noauth.Code != 401 {
		t.Fatalf("expected 401 without key, got %d", noauth.Code)
	}
}

func TestE2E_HostnameCanonicalization(t *testing.T) {
	_, h := newTestServer(t)
	tenantID := seedTenant(t, h, "case-host", "  App.Example.COM. ")
	seedDefinition(t, h, "pg-case")
	if rec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-case",
		"values":        map[string]string{"host": "db", "password": "secret"},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}

	var domains []struct {
		Hostname string `json:"hostname"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantID+"/domains", nil), &domains)
	if len(domains) != 1 || domains[0].Hostname != "app.example.com" {
		t.Fatalf("stored domains = %+v, want canonical hostname", domains)
	}

	token := mintToken(t, h)
	for _, path := range []string{
		"/v1/identify?hostname=APP.EXAMPLE.COM.",
		"/v1/resolve?hostname=App.Example.Com",
		"/v1/resolve/APP.EXAMPLE.COM./resources/pg-case",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 (%s)", path, rec.Code, rec.Body)
		}
	}

	otherTenant := seedTenant(t, h, "case-other", "")
	duplicate := do(t, h, http.MethodPost, "/v1/admin/tenants/"+otherTenant+"/domains",
		map[string]string{"hostname": "app.example.com"})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("case-variant duplicate = %d, want 409 (%s)", duplicate.Code, duplicate.Body)
	}
}

func TestE2E_ResolveOneAndAPIClientLifecycle(t *testing.T) {
	srv, h := newTestServer(t)
	seedDefinition(t, h, "pg-one")
	tenantID := seedTenant(t, h, "one", "one.example.com")
	resource := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-one",
		"values":        map[string]string{"host": "db.one", "password": "one-secret"},
	})
	resourceID := idOf(t, resource)

	client := do(t, h, http.MethodPost, "/v1/admin/api-clients", apiClientCreateBody("lifecycle"))
	var clientBody struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
		Token string `json:"token"`
	}
	mustJSON(t, client, &clientBody)

	var auditCount int
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT count(*) FROM admin_audit_events
		WHERE action = 'api_client.created'
		  AND target_id = $1
		  AND result = 'success'
	`, clientBody.Client.ID).Scan(&auditCount); err != nil {
		t.Fatalf("query API client audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("API client create audit count = %d, want 1", auditCount)
	}

	get := func(path string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if authenticated {
			req.Header.Set("Authorization", "Bearer "+clientBody.Token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	happy := get("/v1/resolve/one.example.com/resources/pg-one", true)
	if happy.Code != http.StatusOK || !strings.Contains(happy.Body.String(), "one-secret") {
		t.Fatalf("resolve one: %d %s", happy.Code, happy.Body)
	}
	if missing := get("/v1/resolve/one.example.com/resources/missing", true); missing.Code != http.StatusNotFound {
		t.Fatalf("missing definition = %d, want 404", missing.Code)
	}
	if missing := get("/v1/resolve/missing.example.com/resources/pg-one", true); missing.Code != http.StatusNotFound {
		t.Fatalf("missing tenant = %d, want 404", missing.Code)
	}
	if unauth := get("/v1/resolve/one.example.com/resources/pg-one", false); unauth.Code != http.StatusUnauthorized {
		t.Fatalf("resolve one without token = %d, want 401", unauth.Code)
	}

	if rec := do(t, h, http.MethodPost, "/v1/admin/api-clients/"+clientBody.Client.ID+"/revoke", nil); rec.Code != http.StatusOK {
		t.Fatalf("revoke client: %d %s", rec.Code, rec.Body)
	}
	if revoked := get("/v1/identify?hostname=one.example.com", true); revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token = %d, want 401", revoked.Code)
	}
	var statusAuditCount int
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT count(*) FROM admin_audit_events
		WHERE action = 'api_client.revoked'
		  AND target_id = $1
		  AND result = 'success'
	`, clientBody.Client.ID).Scan(&statusAuditCount); err != nil {
		t.Fatalf("query API client status audit: %v", err)
	}
	if statusAuditCount != 1 {
		t.Fatalf("API client revoke audit count = %d, want 1", statusAuditCount)
	}
	auditResponse := do(t, h, http.MethodGet,
		"/v1/admin/audit-events?action=api_client.revoked&limit=10", nil)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("list audit events: %d %s", auditResponse.Code, auditResponse.Body)
	}
	var auditPage struct {
		Events []struct {
			Action   string            `json:"action"`
			TargetID string            `json:"target_id"`
			Metadata map[string]string `json:"metadata"`
		} `json:"events"`
	}
	mustJSON(t, auditResponse, &auditPage)
	if len(auditPage.Events) != 1 {
		t.Fatalf("listed revoke events = %d, want 1", len(auditPage.Events))
	}
	for _, event := range auditPage.Events {
		if event.Action != "api_client.revoked" || event.TargetID != clientBody.Client.ID {
			t.Fatalf("unexpected audit event: %+v", event)
		}
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID+"/resources/"+resourceID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete resource: %d %s", rec.Code, rec.Body)
	}
	var resourceCount int
	if err := srv.DB.QueryRow(context.Background(), `SELECT count(*) FROM tenant_resources WHERE id = $1`, resourceID).Scan(&resourceCount); err != nil {
		t.Fatalf("query deleted resource: %v", err)
	}
	if resourceCount != 0 {
		t.Fatalf("deleted resource remains in database")
	}
	var domains []struct {
		ID string `json:"id"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantID+"/domains", nil), &domains)
	if len(domains) != 1 {
		t.Fatalf("domains = %d, want 1", len(domains))
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID+"/domains/"+domains[0].ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete domain: %d %s", rec.Code, rec.Body)
	}
	var remainingDomains []json.RawMessage
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantID+"/domains", nil), &remainingDomains)
	if len(remainingDomains) != 0 {
		t.Fatalf("domain still present after delete: %d", len(remainingDomains))
	}
}

func TestE2E_APIClientScopesAreIndependent(t *testing.T) {
	_, h := newTestServer(t)
	body := apiClientCreateBody("identify-only")
	body["scopes"] = []string{"tenant:identify"}
	created := do(t, h, http.MethodPost, "/v1/admin/api-clients", body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create scoped client: %d %s", created.Code, created.Body)
	}
	var response struct {
		Token string `json:"token"`
	}
	mustJSON(t, created, &response)

	call := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+response.Token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := call("/v1/identify?hostname=missing.example.com"); rec.Code != http.StatusNotFound {
		t.Fatalf("identify status = %d, want handler-level 404", rec.Code)
	}
	if rec := call("/v1/resolve?hostname=missing.example.com"); rec.Code != http.StatusForbidden {
		t.Fatalf("resolve status = %d, want 403 insufficient scope", rec.Code)
	}
}

func TestE2E_APIClientPolicyRotationRevocationAndDelete(t *testing.T) {
	srv, h := newTestServer(t)
	created := do(t, h, http.MethodPost, "/v1/admin/api-clients", apiClientCreateBody("governed"))
	var first struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
		Token string `json:"token"`
	}
	mustJSON(t, created, &first)
	if rec := do(t, h, http.MethodDelete, "/v1/admin/api-clients/"+first.Client.ID, nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete active = %d %s, want 409", rec.Code, rec.Body)
	}
	updatedBody := apiClientCreateBody("governed-renamed")
	updatedBody["scopes"] = []string{"tenant:identify"}
	updatedBody["rpm_limit"] = 600
	if rec := do(t, h, http.MethodPatch, "/v1/admin/api-clients/"+first.Client.ID, updatedBody); rec.Code != http.StatusOK {
		t.Fatalf("update client = %d %s", rec.Code, rec.Body)
	}
	rotated := do(t, h, http.MethodPost, "/v1/admin/api-clients/"+first.Client.ID+"/rotate", map[string]int{"grace_seconds": 60})
	var second struct {
		Token string `json:"token"`
	}
	mustJSON(t, rotated, &second)
	if rotated.Code != http.StatusCreated || second.Token == "" || second.Token == first.Token {
		t.Fatalf("rotation = %d %+v", rotated.Code, second)
	}
	callIdentify := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/v1/identify?hostname=missing.example.com", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := callIdentify(first.Token); code != http.StatusNotFound {
		t.Fatalf("old token during grace = %d, want authenticated 404", code)
	}
	if code := callIdentify(second.Token); code != http.StatusNotFound {
		t.Fatalf("new token = %d, want authenticated 404", code)
	}
	if rec := do(t, h, http.MethodPost, "/v1/admin/api-clients/"+first.Client.ID+"/revoke", nil); rec.Code != http.StatusOK {
		t.Fatalf("revoke = %d %s", rec.Code, rec.Body)
	}
	if code := callIdentify(first.Token); code != http.StatusUnauthorized {
		t.Fatalf("old token after revoke = %d, want 401", code)
	}
	if code := callIdentify(second.Token); code != http.StatusUnauthorized {
		t.Fatalf("new token after revoke = %d, want 401", code)
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/api-clients/"+first.Client.ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete revoked = %d %s", rec.Code, rec.Body)
	}
	var auditCount int
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT count(*) FROM admin_audit_events
		WHERE target_type = 'api_client' AND target_id = $1
		  AND action IN ('api_client.created', 'api_client.policy_updated', 'api_client.rotated', 'api_client.revoked', 'api_client.deleted')
	`, first.Client.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 5 {
		t.Fatalf("lifecycle audit events = %d, want 5", auditCount)
	}
}

func TestE2E_AdminMutationAuditCoverage(t *testing.T) {
	srv, h := newTestServer(t)
	tenantID := seedTenant(t, h, "audit-tenant", "audit.example.com")
	if rec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID, map[string]string{"name": "Audit renamed", "slug": "audit-renamed", "status": "active"}); rec.Code != http.StatusOK {
		t.Fatalf("update tenant: %d %s", rec.Code, rec.Body)
	}
	var domains []struct {
		ID string `json:"id"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantID+"/domains", nil), &domains)
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID+"/domains/"+domains[0].ID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete domain: %d %s", rec.Code, rec.Body)
	}
	definitionID := seedDefinition(t, h, "audit-definition")
	field := do(t, h, http.MethodPost, "/v1/admin/resource-definitions/"+definitionID+"/fields", map[string]any{"key": "unused", "dataType": "string"})
	fieldID := idOf(t, field)
	if rec := do(t, h, http.MethodDelete, "/v1/admin/resource-definitions/"+definitionID+"/fields/"+fieldID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete field: %d %s", rec.Code, rec.Body)
	}
	resource := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "audit-definition", "values": map[string]string{"host": "db", "password": "audit-secret"},
	})
	resourceID := idOf(t, resource)
	if rec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID+"/resources/"+resourceID+"/status", map[string]string{"status": "inactive"}); rec.Code != http.StatusOK {
		t.Fatalf("resource status: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID+"/resources/"+resourceID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete resource: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPut, "/v1/admin/resource-definitions/"+definitionID+"/status", map[string]string{"status": "inactive"}); rec.Code != http.StatusOK {
		t.Fatalf("definition status: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete tenant: %d %s", rec.Code, rec.Body)
	}

	expected := []string{
		"tenant.created", "tenant.updated", "domain.added", "domain.deleted",
		"definition.created", "definition.field_added", "definition.field_deleted",
		"resource.provisioned", "resource.status_changed", "resource.deleted",
		"definition.status_changed", "tenant.deleted",
	}
	for _, action := range expected {
		var count int
		if err := srv.DB.QueryRow(context.Background(), `SELECT count(*) FROM admin_audit_events WHERE action = $1`, action).Scan(&count); err != nil || count == 0 {
			t.Fatalf("audit action %s missing (count=%d err=%v)", action, count, err)
		}
	}
}

func TestE2E_AuditFailureRollsBackAdminMutation(t *testing.T) {
	srv, h := newTestServer(t)
	if _, err := srv.DB.Exec(context.Background(), `DROP TABLE admin_audit_events_default`); err != nil {
		t.Fatalf("remove audit partition: %v", err)
	}
	rec := do(t, h, http.MethodPost, "/v1/admin/tenants", map[string]string{"slug": "must-rollback", "name": "Must rollback"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("create without audit partition = %d %s, want 500", rec.Code, rec.Body)
	}
	var count int
	if err := srv.DB.QueryRow(context.Background(), `SELECT count(*) FROM tenants WHERE slug = 'must-rollback'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("tenant committed without audit event")
	}
}

func TestE2E_DeniedOIDCRequestAuditsDurablePrincipal(t *testing.T) {
	srv, _ := newTestServer(t)
	authStore := adminauth.NewPostgresSessionStore(srv.DB)
	sessions := adminauth.NewSessionManager(authStore, srv.Cryptor, nil, time.Now, 8*time.Hour, 30*time.Minute)
	created, err := sessions.Create(context.Background(), adminauth.SessionIdentity{
		Issuer: "https://id.example.test", Subject: "operator-1", Label: "Ada Operator",
		Roles: []adminauth.Role{adminauth.RoleOperator}, Permissions: []string{"admin.read"},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	srv.ConfigureAdminAuth(adminauth.Config{
		Mode: adminauth.ModeOIDC, CookieName: "tenancit_session", AdminOrigin: "https://tenancit.example.test",
		BreakGlass: adminauth.BreakGlassConfig{Enabled: true, TokenHash: adminauth.HashCredential("break-glass-test-token"), Version: "test-current"},
	}, nil, sessions)
	handler := srv.Routes(nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/audit-events", nil)
	request.AddCookie(&http.Cookie{Name: "tenancit_session", Value: created.Token})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}

	var actorKind, actorIssuer, actorSubject, actorLabel string
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT actor_kind, COALESCE(actor_issuer, ''), actor_subject, COALESCE(actor_label, '')
		FROM admin_audit_events WHERE action = 'admin.request_denied'
		ORDER BY occurred_at DESC, id DESC LIMIT 1
	`).Scan(&actorKind, &actorIssuer, &actorSubject, &actorLabel); err != nil {
		t.Fatalf("query denied audit: %v", err)
	}
	if actorKind != "oidc_user" || actorIssuer != "https://id.example.test" || actorSubject != "operator-1" || actorLabel != "Ada Operator" {
		t.Fatalf("audit actor=%q/%q/%q/%q", actorKind, actorIssuer, actorSubject, actorLabel)
	}

	breakGlassRequest := httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil)
	breakGlassRequest.Header.Set("Authorization", "Bearer break-glass-test-token")
	breakGlassRecorder := httptest.NewRecorder()
	handler.ServeHTTP(breakGlassRecorder, breakGlassRequest)
	if breakGlassRecorder.Code != http.StatusOK {
		t.Fatalf("break-glass status=%d body=%s", breakGlassRecorder.Code, breakGlassRecorder.Body)
	}
	var breakGlassSubject string
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT actor_subject FROM admin_audit_events
		WHERE action = 'break_glass.request_succeeded' ORDER BY occurred_at DESC, id DESC LIMIT 1
	`).Scan(&breakGlassSubject); err != nil {
		t.Fatalf("query break-glass audit: %v", err)
	}
	if breakGlassSubject != "admin-token:test-current" {
		t.Fatalf("break-glass actor=%q", breakGlassSubject)
	}
}

func TestE2E_ETagChangesWithDefinitionFieldsAndValues(t *testing.T) {
	srv, h := newTestServer(t)
	definitionID := seedDefinition(t, h, "pg-etag-content")
	tenantID := seedTenant(t, h, "etag-content", "etag-content.example.com")
	if rec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-etag-content",
		"values":        map[string]string{"host": "db.before", "password": "secret"},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}
	token := mintToken(t, h)
	resolve := func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/resolve?tenantId=etag-content", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	initial := resolve("")
	initialETag := initial.Header().Get("ETag")
	if initial.Code != http.StatusOK || initialETag == "" {
		t.Fatalf("initial resolve: %d etag=%q %s", initial.Code, initialETag, initial.Body)
	}

	if rec := do(t, h, http.MethodPost, "/v1/admin/resource-definitions/"+definitionID+"/fields",
		map[string]any{"key": "database", "label": "Database"}); rec.Code != http.StatusCreated {
		t.Fatalf("add field: %d %s", rec.Code, rec.Body)
	}
	afterField := resolve(initialETag)
	if afterField.Code != http.StatusOK || afterField.Header().Get("ETag") == initialETag {
		t.Fatalf("field mutation did not invalidate ETag: %d old=%q new=%q", afterField.Code, initialETag, afterField.Header().Get("ETag"))
	}
	addedFieldID := definitionFieldID(t, h, definitionID, "database")
	if rec := do(t, h, http.MethodDelete, "/v1/admin/resource-definitions/"+definitionID+"/fields/"+addedFieldID, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete unused field: %d %s", rec.Code, rec.Body)
	}
	fieldETag := afterField.Header().Get("ETag")
	afterFieldDelete := resolve(fieldETag)
	if afterFieldDelete.Code != http.StatusOK || afterFieldDelete.Header().Get("ETag") == fieldETag {
		t.Fatalf("field delete did not invalidate ETag: %d old=%q new=%q", afterFieldDelete.Code, fieldETag, afterFieldDelete.Header().Get("ETag"))
	}

	resources, err := srv.Q.ListTenantResources(ctxTODO(), mustUUID(t, tenantID))
	if err != nil || len(resources) != 1 {
		t.Fatalf("list resources: len=%d err=%v", len(resources), err)
	}
	fields, err := srv.Q.ListFields(ctxTODO(), mustUUID(t, definitionID))
	if err != nil {
		t.Fatalf("list fields: %v", err)
	}
	var hostFieldID string
	for _, field := range fields {
		if field.Key == "host" {
			hostFieldID = field.ID.String()
		}
	}
	if hostFieldID == "" {
		t.Fatal("host field missing")
	}
	changedHost := "db.after"
	if _, err := srv.Q.UpsertResourceValue(ctxTODO(), db.UpsertResourceValueParams{
		TenantResourceID: resources[0].ID,
		ResourceFieldID:  mustUUID(t, hostFieldID),
		ValuePlain:       &changedHost,
	}); err != nil {
		t.Fatalf("upsert resource value: %v", err)
	}
	valueETag := afterFieldDelete.Header().Get("ETag")
	afterValue := resolve(valueETag)
	if afterValue.Code != http.StatusOK || afterValue.Header().Get("ETag") == valueETag {
		t.Fatalf("value mutation did not invalidate ETag: %d old=%q new=%q", afterValue.Code, valueETag, afterValue.Header().Get("ETag"))
	}
	if !strings.Contains(afterValue.Body.String(), changedHost) {
		t.Fatalf("resolved body did not include changed value: %s", afterValue.Body)
	}
}

func mustUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse UUID %q: %v", value, err)
	}
	return id
}

func ctxTODO() context.Context { return context.Background() }
