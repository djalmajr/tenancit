package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestE2E_AdminIdempotencyReplaysCriticalMutations(t *testing.T) {
	server, handler := newTestServer(t)

	tenantKey := uuid.NewString()
	tenantBody := map[string]string{"slug": "idempotent-tenant", "name": "Idempotent tenant"}
	firstTenant := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/tenants", tenantBody, tenantKey)
	secondTenant := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/tenants", tenantBody, tenantKey)
	assertReplay(t, firstTenant, secondTenant)

	mismatch := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/tenants", map[string]string{
		"slug": "different", "name": "Different",
	}, tenantKey)
	if mismatch.Code != http.StatusConflict || !bytes.Contains(mismatch.Body.Bytes(), []byte("idempotency_mismatch")) {
		t.Fatalf("mismatch status=%d body=%s", mismatch.Code, mismatch.Body)
	}

	clientKey := uuid.NewString()
	clientBody := apiClientCreateBody("idempotent-client")
	firstClient := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/api-clients", clientBody, clientKey)
	secondClient := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/api-clients", clientBody, clientKey)
	assertReplay(t, firstClient, secondClient)
	var client createAPIClientResponse
	mustJSON(t, firstClient, &client)

	rotateKey := uuid.NewString()
	rotation := map[string]int{"grace_seconds": 300}
	firstRotate := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/api-clients/"+client.Client.ID.String()+"/rotate", rotation, rotateKey)
	secondRotate := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/api-clients/"+client.Client.ID.String()+"/rotate", rotation, rotateKey)
	assertReplay(t, firstRotate, secondRotate)

	definitionID := seedDefinition(t, handler, "idempotent-pg")
	_ = definitionID
	tenantID := idOf(t, firstTenant)
	resourceKey := uuid.NewString()
	resourceBody := map[string]any{"definitionKey": "idempotent-pg", "values": map[string]string{"host": "db", "password": "secret"}}
	firstResource := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", resourceBody, resourceKey)
	secondResource := doWithIdempotency(t, handler, http.MethodPost, "/v1/admin/tenants/"+tenantID+"/resources", resourceBody, resourceKey)
	assertReplay(t, firstResource, secondResource)

	var tenants, clients, resources, idempotencyRecords int
	if err := server.DB.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM tenants WHERE slug='idempotent-tenant'),
		(SELECT count(*) FROM api_clients WHERE name='idempotent-client'),
		(SELECT count(*) FROM tenant_resources WHERE tenant_id=$1),
		(SELECT count(*) FROM admin_idempotency_records)`, tenantID).
		Scan(&tenants, &clients, &resources, &idempotencyRecords); err != nil {
		t.Fatal(err)
	}
	if tenants != 1 || clients != 1 || resources != 1 || idempotencyRecords < 4 {
		t.Fatalf("tenants=%d clients=%d resources=%d records=%d", tenants, clients, resources, idempotencyRecords)
	}
}

func TestAdminIdempotencyKeyIsRequiredOnlyAfterPayloadValidation(t *testing.T) {
	_, handler := newTestServer(t)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/tenants", bytes.NewBufferString(`{"slug":"valid","name":"Valid"}`))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("idempotency_key_required")) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}

	invalid := httptest.NewRequest(http.MethodPost, "/v1/admin/api-clients", bytes.NewBufferString(`{"name":"","expires_at":"`+time.Now().UTC().Format(time.RFC3339)+`"}`))
	invalid.Header.Set("Authorization", "Bearer "+testAdminToken)
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || bytes.Contains(invalidResponse.Body.Bytes(), []byte("idempotency_key_required")) {
		t.Fatalf("invalid status=%d body=%s", invalidResponse.Code, invalidResponse.Body)
	}
}

func doWithIdempotency(t *testing.T, handler http.Handler, method, path string, body any, key string) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertReplay(t *testing.T, first, second *httptest.ResponseRecorder) {
	t.Helper()
	if first.Code != http.StatusCreated || second.Code != first.Code || second.Header().Get("Idempotency-Replayed") != "true" || !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Fatalf("first=%d/%s second=%d replay=%q/%s", first.Code, first.Body, second.Code, second.Header().Get("Idempotency-Replayed"), second.Body)
	}
}
