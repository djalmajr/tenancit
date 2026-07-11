package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testOperationsToken = "operations-test-token-that-is-long-enough"

func TestE2E_OperationalReportsUseDedicatedCredentialAndIdempotency(t *testing.T) {
	server, _ := newTestServer(t)
	server.SetOperationsReportCredential(testOperationsToken, "agent-v1")
	server.Now = func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }
	handler := server.Routes(nil)
	body := map[string]any{
		"kind": "backup", "source": "postgres-primary", "status": "healthy",
		"occurred_at": "2026-07-11T11:55:00Z", "fresh_for_seconds": 3600,
	}

	unauthorized := operationalReportRequest(t, handler, "", "backup-2026-07-11", body)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", unauthorized.Code, unauthorized.Body)
	}
	created := operationalReportRequest(t, handler, testOperationsToken, "backup-2026-07-11", body)
	if created.Code != http.StatusCreated || created.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("created status=%d cache=%q body=%s", created.Code, created.Header().Get("Cache-Control"), created.Body)
	}
	var first operationalReportView
	mustJSON(t, created, &first)
	duplicate := operationalReportRequest(t, handler, testOperationsToken, "backup-2026-07-11", body)
	var second operationalReportView
	mustJSON(t, duplicate, &second)
	if duplicate.Code != http.StatusOK || first.ID != second.ID {
		t.Fatalf("duplicate status=%d first=%+v second=%+v", duplicate.Code, first, second)
	}
	if _, err := server.DB.Exec(t.Context(), `UPDATE operational_reports SET status='failed' WHERE id=$1`, first.ID); err == nil {
		t.Fatal("database allowed operational report mutation")
	}

	body["status"] = "failed"
	conflict := operationalReportRequest(t, handler, testOperationsToken, "backup-2026-07-11", body)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body)
	}
}

func TestE2E_OperationalHealthNeverTreatsStaleReportAsHealthy(t *testing.T) {
	server, _ := newTestServer(t)
	server.SetOperationsReportCredential(testOperationsToken, "agent-v1")
	server.Now = func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) }
	handler := server.Routes(nil)
	body := map[string]any{
		"kind": "restore", "source": "restore-drill", "status": "healthy",
		"occurred_at": "2026-07-11T11:59:00Z", "fresh_for_seconds": 120,
	}
	if response := operationalReportRequest(t, handler, testOperationsToken, "restore-1", body); response.Code != http.StatusCreated {
		t.Fatalf("create report status=%d body=%s", response.Code, response.Body)
	}

	fresh := do(t, handler, http.MethodGet, "/v1/admin/operational-health", nil)
	var freshOverview operationalHealthView
	mustJSON(t, fresh, &freshOverview)
	if fresh.Code != http.StatusOK || len(freshOverview.Reports) != 1 || freshOverview.Reports[0].EffectiveStatus != "healthy" {
		t.Fatalf("fresh status=%d overview=%+v", fresh.Code, freshOverview)
	}

	server.Now = func() time.Time { return time.Date(2026, 7, 11, 12, 2, 1, 0, time.UTC) }
	stale := do(t, handler, http.MethodGet, "/v1/admin/operational-health", nil)
	var staleOverview operationalHealthView
	mustJSON(t, stale, &staleOverview)
	if staleOverview.Reports[0].EffectiveStatus != "stale" {
		t.Fatalf("stale overview=%+v", staleOverview)
	}
}

func operationalReportRequest(t *testing.T, handler http.Handler, token, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/operations/reports", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
