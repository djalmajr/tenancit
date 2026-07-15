package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Admin routes require the dedicated TENANCIT_ADMIN_TOKEN boundary.
// Mutation captured: removing RequireAdminToken from /v1/admin makes this 200.
func TestAdmin_RequiresAdminToken(t *testing.T) {
	srv, h := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/admin/overview", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401 without admin token", rec.Code)
	}
	var count int
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT count(*) FROM admin_audit_events
		WHERE action = 'admin.request_denied' AND actor_kind = 'unauthenticated' AND http_status = 401
	`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("denied audit count = %d, err=%v", count, err)
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

// A tenant may use the same definition more than once, provided every instance
// has a distinct alias. The alias is the stable consumer-facing identifier.
func TestCreateResource_AllowsRepeatedDefinitionAndRejectsDuplicateAlias(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg")
	tid := seedTenant(t, h, "acme", "app.acme.com")
	first := map[string]any{"definitionKey": "pg", "alias": "pg.primary", "values": map[string]string{"host": "h", "password": "p"}}
	second := map[string]any{"definitionKey": "pg", "alias": "pg.analytics", "values": map[string]string{"host": "h2", "password": "p2"}}

	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", first); rec.Code != 201 {
		t.Fatalf("first create: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", second); rec.Code != 201 {
		t.Fatalf("second definition instance: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "POST", "/v1/admin/tenants/"+tid+"/resources", first)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate alias got %d, want 409 (%s)", rec.Code, rec.Body)
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
	srv, h := newTestServer(t)
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
	if got := revealed.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("reveal Cache-Control = %q, want private, no-store", got)
	}
	var rs []resourceView
	mustJSON(t, revealed, &rs)
	if pw := findField(rs, "password"); pw != "topsecret" {
		t.Fatalf("reveal should decrypt password, got %q", pw)
	}
	var auditMetadata string
	if err := srv.DB.QueryRow(context.Background(), `
		SELECT metadata::text FROM admin_audit_events
		WHERE action = 'secret.revealed' AND target_id = $1
		ORDER BY occurred_at DESC LIMIT 1
	`, tid).Scan(&auditMetadata); err != nil {
		t.Fatalf("query reveal audit: %v", err)
	}
	if strings.Contains(auditMetadata, "topsecret") {
		t.Fatalf("reveal audit leaked secret: %s", auditMetadata)
	}
}

func TestSharedResource_InheritsOverridesAndDuplicatesIndependentSnapshot(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg-shared")
	tenantID := seedTenant(t, h, "shared", "shared.example.com")

	baseRec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-shared",
		"alias":         "pg.base",
		"values":        map[string]string{"host": "db.internal", "password": "secret-v1"},
	})
	if baseRec.Code != http.StatusCreated {
		t.Fatalf("create base: %d %s", baseRec.Code, baseRec.Body)
	}
	baseID := idOf(t, baseRec)

	linkedRec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey":    "pg-shared",
		"alias":            "pg.agility",
		"sourceResourceId": baseID,
		"values":           map[string]string{"host": "agility-db.internal"},
	})
	if linkedRec.Code != http.StatusCreated {
		t.Fatalf("create linked: %d %s", linkedRec.Code, linkedRec.Body)
	}
	linkedID := idOf(t, linkedRec)

	resources := listRevealedResources(t, h, tenantID)
	linked := findResource(resources, "pg.agility")
	if linked == nil || !linked.Linked {
		t.Fatalf("linked resource = %+v, want linked", linked)
	}
	if host := findResourceField(linked, "host"); host == nil || host.Value != "agility-db.internal" || host.Origin != "local" {
		t.Fatalf("linked host = %+v, want local override", host)
	}
	if password := findResourceField(linked, "password"); password == nil || password.Value != "secret-v1" || password.Origin != "inherited" {
		t.Fatalf("linked password = %+v, want inherited", password)
	}

	if rec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID+"/resources/"+baseID+"/fields/password", map[string]string{"value": "secret-v2"}); rec.Code != http.StatusNoContent {
		t.Fatalf("update base: %d %s", rec.Code, rec.Body)
	}
	linked = findResource(listRevealedResources(t, h, tenantID), "pg.agility")
	if got := findResourceField(linked, "password"); got == nil || got.Value != "secret-v2" || got.Origin != "inherited" {
		t.Fatalf("updated inherited password = %+v", got)
	}

	duplicateRec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources/"+linkedID+"/duplicate", map[string]string{"alias": "pg.snapshot"})
	if duplicateRec.Code != http.StatusCreated {
		t.Fatalf("duplicate linked resource: %d %s", duplicateRec.Code, duplicateRec.Body)
	}
	if rec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID+"/resources/"+baseID+"/fields/password", map[string]string{"value": "secret-v3"}); rec.Code != http.StatusNoContent {
		t.Fatalf("update base again: %d %s", rec.Code, rec.Body)
	}

	resources = listRevealedResources(t, h, tenantID)
	linked = findResource(resources, "pg.agility")
	snapshot := findResource(resources, "pg.snapshot")
	if got := findResourceField(linked, "password"); got == nil || got.Value != "secret-v3" {
		t.Fatalf("linked resource stopped inheriting: %+v", got)
	}
	if snapshot == nil || snapshot.Linked {
		t.Fatalf("snapshot = %+v, want independent", snapshot)
	}
	if got := findResourceField(snapshot, "password"); got == nil || got.Value != "secret-v2" || got.Origin != "local" {
		t.Fatalf("snapshot password = %+v, want materialized v2", got)
	}

	if rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantID+"/resources/"+baseID, nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete source with dependents = %d, want 409 (%s)", rec.Code, rec.Body)
	}
}

func listRevealedResources(t *testing.T, h http.Handler, tenantID string) []resourceView {
	t.Helper()
	rec := do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantID+"/resources?reveal=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list resources: %d %s", rec.Code, rec.Body)
	}
	var resources []resourceView
	mustJSON(t, rec, &resources)
	return resources
}

type fieldView struct {
	Key      string `json:"key"`
	IsSecret bool   `json:"isSecret"`
	Value    string `json:"value"`
	Origin   string `json:"origin"`
}

type resourceView struct {
	ID     string      `json:"id"`
	Alias  string      `json:"alias"`
	Linked bool        `json:"linked"`
	Fields []fieldView `json:"fields"`
}

func findResource(resources []resourceView, alias string) *resourceView {
	for i := range resources {
		if resources[i].Alias == alias {
			return &resources[i]
		}
	}
	return nil
}

func findResourceField(resource *resourceView, key string) *fieldView {
	if resource == nil {
		return nil
	}
	for i := range resource.Fields {
		if resource.Fields[i].Key == key {
			return &resource.Fields[i]
		}
	}
	return nil
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

func TestUpdateTenant_RejectsInvalidStatus(t *testing.T) {
	_, h := newTestServer(t)
	tenantID := seedTenant(t, h, "acme", "")

	rec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID,
		map[string]string{"name": "Acme", "slug": "acme", "status": "paused"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "status must be active|inactive") {
		t.Fatalf("unexpected invalid-status body: %s", rec.Body)
	}
}

func TestTenantMutations_RejectInvalidSlug(t *testing.T) {
	_, h := newTestServer(t)

	created := do(t, h, http.MethodPost, "/v1/admin/tenants", map[string]string{
		"name": "Invalid", "slug": "invalid|slug",
	})
	if created.Code != http.StatusBadRequest || !strings.Contains(created.Body.String(), "invalid tenant slug") {
		t.Fatalf("create invalid slug = %d, want 400 (%s)", created.Code, created.Body)
	}

	tenantID := seedTenant(t, h, "valid-slug", "")
	updated := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantID, map[string]string{
		"name": "Invalid", "slug": "invalid_slug", "status": "active",
	})
	if updated.Code != http.StatusBadRequest || !strings.Contains(updated.Body.String(), "invalid tenant slug") {
		t.Fatalf("update invalid slug = %d, want 400 (%s)", updated.Code, updated.Body)
	}
}

func TestDefinitionMutations_RejectInvalidKeys(t *testing.T) {
	_, h := newTestServer(t)

	definition := do(t, h, http.MethodPost, "/v1/admin/resource-definitions", map[string]string{
		"key": "invalid.key", "name": "Invalid",
	})
	if definition.Code != http.StatusBadRequest || !strings.Contains(definition.Body.String(), "invalid definition key") {
		t.Fatalf("invalid definition key = %d, want 400 (%s)", definition.Code, definition.Body)
	}

	definitionID := seedDefinition(t, h, "valid")
	field := do(t, h, http.MethodPost, "/v1/admin/resource-definitions/"+definitionID+"/fields", map[string]string{
		"key": "invalid.field", "label": "Invalid",
	})
	if field.Code != http.StatusBadRequest || !strings.Contains(field.Body.String(), "invalid field key") {
		t.Fatalf("invalid field key = %d, want 400 (%s)", field.Code, field.Body)
	}
}

func TestAdmin_APIClientResponsesOmitKeyHash(t *testing.T) {
	_, h := newTestServer(t)

	created := do(t, h, http.MethodPost, "/v1/admin/api-clients", apiClientCreateBody("edge"))
	if created.Code != http.StatusCreated {
		t.Fatalf("create client: %d %s", created.Code, created.Body)
	}
	if got := created.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("create client Cache-Control = %q, want private, no-store", got)
	}
	var createBody struct {
		Client map[string]any `json:"client"`
		Token  string         `json:"token"`
	}
	mustJSON(t, created, &createBody)
	if !strings.HasPrefix(createBody.Token, "tnc_") {
		t.Fatalf("one-time token missing or malformed")
	}
	assertNoKeyHash(t, createBody.Client)
	clientID, _ := createBody.Client["id"].(string)
	if clientID == "" {
		t.Fatalf("client id missing: %s", created.Body)
	}

	listed := do(t, h, http.MethodGet, "/v1/admin/api-clients", nil)
	var clients []map[string]any
	mustJSON(t, listed, &clients)
	if len(clients) != 1 {
		t.Fatalf("list clients = %d, want 1", len(clients))
	}
	assertNoKeyHash(t, clients[0])

	updated := do(t, h, http.MethodPost, "/v1/admin/api-clients/"+clientID+"/revoke", nil)
	if updated.Code != http.StatusOK {
		t.Fatalf("revoke client: %d %s", updated.Code, updated.Body)
	}
	var updateBody map[string]any
	mustJSON(t, updated, &updateBody)
	assertNoKeyHash(t, updateBody)
}

func TestCreateAPIClient_RequiresPolicy(t *testing.T) {
	_, h := newTestServer(t)
	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{name: "scope", body: map[string]any{"name": "bad", "rpm_limit": 300, "expires_at": time.Now().Add(24 * time.Hour)}, want: "invalid_scope"},
		{name: "rpm", body: map[string]any{"name": "bad", "scopes": []string{"tenant:identify"}, "expires_at": time.Now().Add(24 * time.Hour)}, want: "invalid_rpm"},
		{name: "expiration", body: map[string]any{"name": "bad", "scopes": []string{"tenant:identify"}, "rpm_limit": 300}, want: "invalid_expiration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/v1/admin/api-clients", tt.body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("response = %d %s, want 400 %s", rec.Code, rec.Body, tt.want)
			}
		})
	}
}

func assertNoKeyHash(t *testing.T, client map[string]any) {
	t.Helper()
	if _, exists := client["key_hash"]; exists {
		t.Fatalf("API client response exposed key_hash: %v", client)
	}
	for _, key := range []string{"id", "name", "status", "created_at"} {
		if _, exists := client[key]; !exists {
			t.Fatalf("API client response missing %q: %v", key, client)
		}
	}
}

func TestAdmin_OversizedJSONBodyReturns413(t *testing.T) {
	_, h := newTestServer(t)
	rec := do(t, h, http.MethodPost, "/v1/admin/tenants", map[string]string{
		"slug": "oversized",
		"name": strings.Repeat("x", (1<<20)+1),
	})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body = %d, want 413 (response bytes=%d)", rec.Code, rec.Body.Len())
	}
	if rec.Body.String() != "{\"error\":\"request body too large\"}\n" {
		t.Fatalf("unexpected oversized-body response: %s", rec.Body)
	}
}

func TestAdmin_InternalErrorsAreOpaque(t *testing.T) {
	srv, h := newTestServer(t)
	srv.DB.Close()

	rec := do(t, h, http.MethodGet, "/v1/admin/tenants", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("closed DB = %d, want 500 (%s)", rec.Code, rec.Body)
	}
	if rec.Body.String() != "{\"error\":\"internal error\"}\n" {
		t.Fatalf("internal error leaked implementation details: %s", rec.Body)
	}
}

func TestSetResourceStatus_MissingResourceReturns404(t *testing.T) {
	_, h := newTestServer(t)
	tenantID := seedTenant(t, h, "acme", "")
	rec := do(t, h, http.MethodPut,
		"/v1/admin/tenants/"+tenantID+"/resources/00000000-0000-4000-8000-000000000001/status",
		map[string]string{"status": "inactive"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing resource status = %d, want 404 (%s)", rec.Code, rec.Body)
	}
}

func TestNestedDomainMutationRequiresMatchingTenant(t *testing.T) {
	_, h := newTestServer(t)
	tenantA := seedTenant(t, h, "a", "a.example.com")
	tenantB := seedTenant(t, h, "b", "")
	var domains []struct {
		ID string `json:"id"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantA+"/domains", nil), &domains)
	if len(domains) != 1 {
		t.Fatalf("domains = %d, want 1", len(domains))
	}

	updateRec := do(t, h, http.MethodPut, "/v1/admin/tenants/"+tenantB+"/domains/"+domains[0].ID,
		map[string]string{"hostname": "stolen.example.com"})
	if updateRec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant domain update = %d, want 404 (%s)", updateRec.Code, updateRec.Body)
	}
	rec := do(t, h, http.MethodDelete, "/v1/admin/tenants/"+tenantB+"/domains/"+domains[0].ID, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant domain delete = %d, want 404 (%s)", rec.Code, rec.Body)
	}
	var after []struct {
		Hostname string `json:"hostname"`
	}
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantA+"/domains", nil), &after)
	if len(after) != 1 || after[0].Hostname != "a.example.com" {
		t.Fatalf("cross-tenant mutation changed domain; remaining=%+v", after)
	}
}

func TestNestedResourceMutationsRequireMatchingTenant(t *testing.T) {
	_, h := newTestServer(t)
	seedDefinition(t, h, "pg-parent")
	tenantA := seedTenant(t, h, "a", "")
	tenantB := seedTenant(t, h, "b", "")
	created := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantA+"/resources", map[string]any{
		"definitionKey": "pg-parent",
		"values":        map[string]string{"host": "db", "password": "secret"},
	})
	resourceID := idOf(t, created)
	identityRec := do(t, h, http.MethodPatch,
		"/v1/admin/tenants/"+tenantB+"/resources/"+resourceID,
		map[string]string{"name": "Other", "alias": "pg.other"})
	if identityRec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant identity update = %d, want 404 (%s)", identityRec.Code, identityRec.Body)
	}

	statusRec := do(t, h, http.MethodPut,
		"/v1/admin/tenants/"+tenantB+"/resources/"+resourceID+"/status",
		map[string]string{"status": "inactive"})
	if statusRec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant status = %d, want 404 (%s)", statusRec.Code, statusRec.Body)
	}
	deleteRec := do(t, h, http.MethodDelete,
		"/v1/admin/tenants/"+tenantB+"/resources/"+resourceID, nil)
	if deleteRec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant delete = %d, want 404 (%s)", deleteRec.Code, deleteRec.Body)
	}
	var remaining []json.RawMessage
	mustJSON(t, do(t, h, http.MethodGet, "/v1/admin/tenants/"+tenantA+"/resources", nil), &remaining)
	if len(remaining) != 1 {
		t.Fatalf("cross-tenant mutation removed resource; remaining=%d", len(remaining))
	}
}

func TestNestedFieldDeleteRequiresMatchingDefinition(t *testing.T) {
	_, h := newTestServer(t)
	definitionA := seedDefinition(t, h, "def-a")
	definitionB := seedDefinition(t, h, "def-b")
	fieldID := definitionFieldID(t, h, definitionA, "password")

	rec := do(t, h, http.MethodDelete,
		"/v1/admin/resource-definitions/"+definitionB+"/fields/"+fieldID, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-definition field delete = %d, want 404 (%s)", rec.Code, rec.Body)
	}
	if got := definitionFieldID(t, h, definitionA, "password"); got != fieldID {
		t.Fatalf("cross-definition delete removed field: got %q, want %q", got, fieldID)
	}
}

func TestDeleteField_InUseReturnsStable409(t *testing.T) {
	_, h := newTestServer(t)
	definitionID := seedDefinition(t, h, "pg-in-use")
	tenantID := seedTenant(t, h, "acme", "")
	if rec := do(t, h, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"definitionKey": "pg-in-use",
		"values":        map[string]string{"host": "db", "password": "secret"},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create resource: %d %s", rec.Code, rec.Body)
	}
	fieldID := definitionFieldID(t, h, definitionID, "password")

	rec := do(t, h, http.MethodDelete,
		"/v1/admin/resource-definitions/"+definitionID+"/fields/"+fieldID, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete in-use field = %d, want 409 (%s)", rec.Code, rec.Body)
	}
	if rec.Body.String() != "{\"error\":\"field is in use by tenant resources\"}\n" {
		t.Fatalf("unexpected in-use response: %s", rec.Body)
	}
}

func definitionFieldID(t *testing.T, h http.Handler, definitionID, key string) string {
	t.Helper()
	var detail struct {
		Fields []struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"fields"`
	}
	rec := do(t, h, http.MethodGet, "/v1/admin/resource-definitions/"+definitionID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get definition: %d %s", rec.Code, rec.Body)
	}
	mustJSON(t, rec, &detail)
	for _, field := range detail.Fields {
		if field.Key == key {
			return field.ID
		}
	}
	t.Fatalf("field %q not found in definition %s", key, definitionID)
	return ""
}
