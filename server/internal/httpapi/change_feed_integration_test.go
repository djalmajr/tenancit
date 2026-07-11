package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func createFeedClient(t *testing.T, handler http.Handler, name string, scopes []string) string {
	t.Helper()
	response := do(t, handler, http.MethodPost, "/v1/admin/api-clients", map[string]any{
		"name": name, "scopes": scopes, "rpm_limit": 300,
		"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("create client status=%d body=%s", response.Code, response.Body)
	}
	var body struct {
		Token string `json:"token"`
	}
	mustJSON(t, response, &body)
	return body.Token
}

func getFeed(t *testing.T, handler http.Handler, token, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestE2E_ChangeFeedRequiresScopeAndPaginatesWithoutSecrets(t *testing.T) {
	_, handler := newTestServer(t)
	feedToken := createFeedClient(t, handler, "feed-reader", []string{"events:read"})
	identifyToken := createFeedClient(t, handler, "identify-only", []string{"tenant:identify"})
	for index := 0; index < 3; index++ {
		response := do(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{"slug": "feed-" + string(rune('a'+index)), "name": "Private name"})
		if response.Code != http.StatusCreated {
			t.Fatalf("tenant status=%d body=%s", response.Code, response.Body)
		}
	}

	denied := getFeed(t, handler, identifyToken, "/v1/events")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied status=%d body=%s", denied.Code, denied.Body)
	}
	first := getFeed(t, handler, feedToken, "/v1/events?limit=2")
	if first.Code != http.StatusOK || first.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body)
	}
	var page1 changeFeedResponse
	mustJSON(t, first, &page1)
	if len(page1.Events) != 2 || page1.NextCursor == "" {
		t.Fatalf("page1=%+v", page1)
	}
	second := getFeed(t, handler, feedToken, "/v1/events?limit=200&cursor="+url.QueryEscape(page1.NextCursor))
	var page2 changeFeedResponse
	mustJSON(t, second, &page2)
	if len(page2.Events) < 1 {
		t.Fatalf("page2=%+v", page2)
	}
	all := append(append([]changeFeedEvent{}, page1.Events...), page2.Events...)
	tenantEvents := 0
	for _, event := range all {
		if event.Type == "tenancit.tenant.created" {
			tenantEvents++
		}
	}
	if tenantEvents != 3 {
		t.Fatalf("tenant events=%d all=%+v", tenantEvents, all)
	}
	combined, _ := json.Marshal(all)
	if strings.Contains(string(combined), "Private name") || strings.Contains(strings.ToLower(string(combined)), "password") {
		t.Fatalf("feed leaked sensitive fields: %s", combined)
	}
}
