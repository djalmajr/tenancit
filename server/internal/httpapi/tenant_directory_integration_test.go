package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// GET /v1/tenants: directory identities only, gated by the tenant:list scope.
func TestE2E_TenantDirectoryRequiresScopeAndExposesNoSecrets(t *testing.T) {
	_, handler := newTestServer(t)
	listToken := createFeedClient(t, handler, "directory-reader", []string{"tenant:list"})
	identifyToken := createFeedClient(t, handler, "identify-only-directory", []string{"tenant:identify"})

	for _, slug := range []string{"dir-a", "dir-b"} {
		response := do(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{"slug": slug, "name": "Tenant " + slug})
		if response.Code != http.StatusCreated {
			t.Fatalf("tenant status=%d body=%s", response.Code, response.Body)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+listToken)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", recorder.Code, recorder.Body)
	}
	var entries []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 tenants, got %d", len(entries))
	}
	for _, entry := range entries {
		for _, forbidden := range []string{"resources", "values", "id", "secret"} {
			if _, present := entry[forbidden]; present {
				t.Fatalf("directory entry leaked field %q: %v", forbidden, entry)
			}
		}
		if entry["slug"] == "" || entry["status"] == "" {
			t.Fatalf("directory entry missing identity fields: %v", entry)
		}
	}
	if !strings.Contains(recorder.Body.String(), "dir-a") {
		t.Fatalf("expected dir-a in directory: %s", recorder.Body)
	}

	denied := httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	denied.Header.Set("Authorization", "Bearer "+identifyToken)
	deniedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deniedRecorder, denied)
	if deniedRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without tenant:list, got %d body=%s", deniedRecorder.Code, deniedRecorder.Body)
	}
}
