package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djalmajr/konvario/server/internal/crypto"
	"github.com/djalmajr/konvario/server/internal/testsupport"
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
	rec := do(t, h, "POST", "/v1/admin/api-clients", map[string]string{"name": "c"})
	if rec.Code != 201 {
		t.Fatalf("api client: %d %s", rec.Code, rec.Body)
	}
	var out struct {
		Token string `json:"token"`
	}
	mustJSON(t, rec, &out)
	return out.Token
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

	// Resolve without an API key must be rejected (RN-09).
	// Mutation captured: removing RequireAPIKey middleware would make this 200.
	rrec = httptest.NewRecorder()
	h.ServeHTTP(rrec, httptest.NewRequest("GET", "/v1/resolve?hostname=e2e.example.com", nil))
	if rrec.Code != 401 {
		t.Fatalf("expected 401 without key, got %d", rrec.Code)
	}
}

func ctxTODO() context.Context { return context.Background() }
