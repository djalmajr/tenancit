package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// PATCH /v1/tenants/{slug}/resources/{alias}/values/{fieldKey}: non-secret
// fields only, gated by resource:write.
func TestE2E_ConsumerWriteValueNonSecretOnly(t *testing.T) {
	_, handler := newTestServer(t)
	writeToken := createFeedClient(t, handler, "writer", []string{"resource:write", "resource:resolve"})
	readToken := createFeedClient(t, handler, "reader-only", []string{"resource:resolve"})

	definition := do(t, handler, http.MethodPost, "/v1/admin/resource-definitions", map[string]string{"key": "branding", "name": "Branding"})
	if definition.Code != http.StatusCreated {
		t.Fatalf("definition status=%d body=%s", definition.Code, definition.Body)
	}
	definitionID := jsonField(t, definition.Body.Bytes(), "id")
	for _, field := range []map[string]any{
		{"key": "brand_name", "label": "Brand", "isSecret": false},
		{"key": "api_secret", "label": "Secret", "isSecret": true},
	} {
		response := do(t, handler, http.MethodPost, "/v1/admin/resource-definitions/"+definitionID+"/fields", field)
		if response.Code != http.StatusCreated {
			t.Fatalf("field status=%d body=%s", response.Code, response.Body)
		}
	}

	tenant := do(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{"slug": "writer-tenant", "name": "Writer"})
	if tenant.Code != http.StatusCreated {
		t.Fatalf("tenant status=%d body=%s", tenant.Code, tenant.Body)
	}
	tenantID := jsonField(t, tenant.Body.Bytes(), "id")
	resource := do(t, handler, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", map[string]any{
		"name": "branding.writer", "alias": "branding.writer", "definitionKey": "branding",
		"values": map[string]string{"brand_name": "Antes", "api_secret": "s3cr3t"},
	})
	if resource.Code != http.StatusCreated {
		t.Fatalf("resource status=%d body=%s", resource.Code, resource.Body)
	}

	patch := func(token, fieldKey, value string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPatch,
			"/v1/tenants/writer-tenant/resources/branding.writer/values/"+fieldKey,
			strings.NewReader(`{"value":"`+value+`"}`))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}

	if response := patch(writeToken, "brand_name", "Depois"); response.Code != http.StatusNoContent {
		t.Fatalf("non-secret write status=%d body=%s", response.Code, response.Body)
	}
	if response := patch(writeToken, "api_secret", "hack"); response.Code != http.StatusForbidden {
		t.Fatalf("secret write should be 403, got %d body=%s", response.Code, response.Body)
	}
	if response := patch(readToken, "brand_name", "x"); response.Code != http.StatusForbidden {
		t.Fatalf("write without scope should be 403, got %d body=%s", response.Code, response.Body)
	}
}

func jsonField(t *testing.T, body []byte, key string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode %s: %v", key, err)
	}
	value, _ := payload[key].(string)
	if value == "" {
		t.Fatalf("missing field %q in %s", key, body)
	}
	return value
}
