package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
)

type healthProbeStub struct {
	name     string
	critical bool
	err      error
}

func (p healthProbeStub) Name() string                { return p.name }
func (p healthProbeStub) Critical() bool              { return p.critical }
func (p healthProbeStub) Check(context.Context) error { return p.err }

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	NewRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want %q", body["status"], "ok")
	}

	wantHeaders := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
	}
	for name, want := range wantHeaders {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
	csp := rec.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"script-src 'self'", "frame-ancestors 'none'", "object-src 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP %q is missing %q", csp, directive)
		}
	}
}

func TestReadyzReturnsSanitizedComponentsAndUnavailableStatus(t *testing.T) {
	server := &Server{
		Now: func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) },
		ReadinessProbes: []telemetry.Probe{
			healthProbeStub{name: "postgres", critical: true},
			healthProbeStub{name: "valkey", critical: true, err: errors.New("redis://secret@valkey.internal")},
		},
	}
	recorder := httptest.NewRecorder()
	server.Routes(nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}
	if strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "internal") {
		t.Fatalf("readiness leaked probe error: %s", recorder.Body)
	}
	var body telemetry.ReadinessReport
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != telemetry.StatusUnavailable || len(body.Components) != 2 {
		t.Fatalf("body=%+v", body)
	}
}
