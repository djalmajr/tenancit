package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Admin routes require the dedicated TENANCIT_ADMIN_TOKEN boundary.
// Mutation captured: removing RequireAdminToken from /v1/admin makes this 200.
func TestAdmin_RequiresAdminToken(t *testing.T) {
	_, h := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/admin/overview", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401 without admin token", rec.Code)
	}
}

// RN-03: required field missing on resource create must be rejected (400),
// before any row is written.
// Mutation captured: deleting the `if f.Required { ... }` loop flips this to 201.
func TestCreateResource_MissingRequired_400(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")

	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", map[string]any{
		"definitionKey": "pg",
		"values":        map[string]string{"host": "h"}, // password missing
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 (%s)", rec.Code, rec.Body)
	}
}

// RN-01: a second ACTIVE resource of the same definition for one tenant is a conflict.
// Mutation captured: dropping the partial unique index OR swallowing the insert
// error would let this return 201.
func TestCreateResource_DuplicateActive_409(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	body := map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "p"}}

	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", body); rec.Code != 201 {
		t.Fatalf("first create: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second active create got %d, want 409 (%s)", rec.Code, rec.Body)
	}
}

// RN-08: cannot provision a resource of an inactive definition.
// Mutation captured: removing the `def.Status != "active"` guard returns 201.
func TestCreateResource_InactiveDefinition_400(t *testing.T) {
	_, h := newTestServer(t)
	defID := seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	if rec := do(t, h, "PUT", "/v1/admin/resource-definitions/"+defID+"/status",
		map[string]string{"status": "inactive"}); rec.Code != 200 {
		t.Fatalf("deactivate def: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources",
		map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "p"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 for inactive def (%s)", rec.Code, rec.Body)
	}
}

// RN-06: admin resource listing masks secrets by default and reveals with ?reveal=true.
// Mutation captured: making presentValue ignore the reveal flag (always cleartext)
// fails the masked assertion; always-mask fails the reveal assertion.
func TestListTenantResources_MaskAndReveal(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources",
		map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "topsecret"}}); rec.Code != 201 {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}

	// default → masked
	masked := do(t, h, "GET", "/v1/admin/tenants/"+tid+"/resources", nil)
	var ms []resourceView
	mustJSON(t, masked, &ms)
	if pw := findField(ms, "password"); pw == "topsecret" || pw == "" {
		t.Fatalf("password should be masked by default, got %q", pw)
	}
	if host := findField(ms, "host"); host != "h" {
		t.Fatalf("non-secret host should be cleartext, got %q", host)
	}

	// ?reveal=true → cleartext
	revealed := do(t, h, "GET", "/v1/admin/tenants/"+tid+"/resources?reveal=true", nil)
	var rs []resourceView
	mustJSON(t, revealed, &rs)
	if pw := findField(rs, "password"); pw != "topsecret" {
		t.Fatalf("reveal should decrypt password, got %q", pw)
	}
}

type fieldView struct {
	Key      string `json:"key"`
	IsSecret bool   `json:"isSecret"`
	Value    string `json:"value"`
}

type resourceView struct {
	Fields []fieldView `json:"fields"`
}

func findField(resources []resourceView, key string) string {
	for _, r := range resources {
		for _, f := range r.Fields {
			if f.Key == key {
				return f.Value
			}
		}
	}
	return ""
}

// Overview counts must reflect real data (the bug that showed 0 tenants).
// Mutation captured: an overview handler that returns zeros / isn't mounted
// (falls through to SPA) fails the count assertions.
func TestOverview_Counts(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	_ = do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources",
		map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "p"}})
	seedTenant(t, h, "globex", "")

	rec := do(t, h, "GET", "/v1/admin/overview", nil)
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("overview must be JSON, got content-type %q body=%s", ct, rec.Body)
	}
	var o struct {
		Tenants       int `json:"tenants"`
		ActiveTenants int `json:"activeTenants"`
		Domains       int `json:"domains"`
		Resources     int `json:"resources"`
		Definitions   int `json:"definitions"`
	}
	mustJSON(t, rec, &o)
	if o.Tenants != 2 || o.Domains != 1 || o.Resources != 1 || o.Definitions != 1 {
		t.Fatalf("overview counts wrong: %+v", o)
	}
}

// Resolver excludes inactive resources (only active config is served).
// Mutation captured: switching ListActiveResourcesByTenant to ListTenantResources
// (dropping the status='active' filter) makes the disabled resource appear.
func TestResolve_ExcludesInactiveResource(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources",
		map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "p"}})
	rid := idOf(t, rec)
	if r := do(t, h, "PUT", "/v1/admin/tenants/"+tid+"/resources/"+rid+"/status",
		map[string]string{"status": "inactive"}); r.Code != 200 {
		t.Fatalf("deactivate resource: %d %s", r.Code, r.Body)
	}

	token := mintToken(t, h)
	req := httptest.NewRequest("GET", "/v1/resolve?hostname=app.acme.com", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rrec := httptest.NewRecorder()
	h.ServeHTTP(rrec, req)
	if rrec.Code != 200 {
		t.Fatalf("resolve: %d %s", rrec.Code, rrec.Body)
	}
	var resolved struct {
		Resources []struct {
			DefinitionKey string `json:"definitionKey"`
		} `json:"resources"`
	}
	mustJSON(t, rrec, &resolved)
	if len(resolved.Resources) != 0 {
		t.Fatalf("inactive resource leaked into resolve: %+v", resolved.Resources)
	}
}

// Unknown hostname resolves to 404, not a 200 with empty tenant.
// Mutation captured: ignoring the GetTenantByHostname error would 200 here.
func TestResolve_UnknownHostname_404(t *testing.T) {
	_, h := newTestServer(t)
	token := mintToken(t, h)
	req := httptest.NewRequest("GET", "/v1/resolve?hostname=nope.example.com", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rrec := httptest.NewRecorder()
	h.ServeHTTP(rrec, req)
	if rrec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404 for unknown hostname (%s)", rrec.Code, rrec.Body)
	}
}

// Duplicate hostname across tenants is rejected (hostname is globally unique → 1:1).
// Mutation captured: dropping the UNIQUE on tenant_domains.hostname returns 201.
func TestAddDomain_DuplicateHostname_Conflict(t *testing.T) {
	_, h := newTestServer(t)
	a := seedTenant(t, h, "acme", "shared.example.com")
	_ = a
	b := seedTenant(t, h, "globex", "")
	rec := do(t, h, "POST", "/v1/admin/tenants/"+b+"/domains", map[string]string{"hostname": "shared.example.com"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d, want 409 for duplicate hostname (%s)", rec.Code, rec.Body)
	}
}

// Deleting a tenant returns 204, cascades to its domains/resources, and is 404
// thereafter (and on a missing id).
// Mutation captured: a DeleteTenant that ignores RowsAffected returns 204 on a
// missing tenant; a delete that doesn't cascade leaves the domain resolvable.
func TestDeleteTenant_CascadesAnd404(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "delme", "delme.example.com")
	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources",
		map[string]any{"definitionKey": "pg", "values": map[string]string{"host": "h", "password": "p"}}); rec.Code != 201 {
		t.Fatalf("seed resource: %d %s", rec.Code, rec.Body)
	}
	token := mintToken(t, h)
	resolve := func() int {
		req := httptest.NewRequest("GET", "/v1/resolve?hostname=delme.example.com", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr.Code
	}

	if c := resolve(); c != http.StatusOK {
		t.Fatalf("pre-delete resolve got %d, want 200", c)
	}
	if rec := do(t, h, "DELETE", "/v1/admin/tenants/"+tid, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete got %d, want 204 (%s)", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/v1/admin/tenants/"+tid, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete got %d, want 404 (%s)", rec.Code, rec.Body)
	}
	if c := resolve(); c != http.StatusNotFound {
		t.Fatalf("post-delete resolve got %d, want 404 (domain should cascade)", c)
	}
	if rec := do(t, h, "DELETE", "/v1/admin/tenants/"+tid, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete got %d, want 404 (%s)", rec.Code, rec.Body)
	}
}

// Tenant update persists changes (name/slug/status).
// Mutation captured: a no-op UpdateTenant (returns input unchanged) fails the reread.
func TestUpdateTenant_Persists(t *testing.T) {
	_, h := newTestServer(t)
	tid := seedTenant(t, h, "acme", "")
	if rec := do(t, h, "PUT", "/v1/admin/tenants/"+tid,
		map[string]string{"name": "Acme Renamed", "slug": "acme2", "status": "inactive"}); rec.Code != 200 {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "GET", "/v1/admin/tenants/"+tid, nil)
	var got struct{ Name, Slug, Status string }
	mustJSON(t, rec, &got)
	if got.Name != "Acme Renamed" || got.Slug != "acme2" || got.Status != "inactive" {
		t.Fatalf("update not persisted: %+v", got)
	}
}
