package rewrap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHTTPReporterUsesDedicatedCredentialAndStableMetadata(t *testing.T) {
	token := "rewrap-operations-token-that-is-long-enough"
	var authorization, idempotency string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		idempotency = request.Header.Get("Idempotency-Key")
		body, _ := io.ReadAll(request.Body)
		if strings.Contains(string(body), "plaintext") || strings.Contains(string(body), "cipher") || strings.Contains(string(body), "nonce") {
			t.Error("report included sensitive fields")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	reporter, err := NewHTTPReporter(server.URL+"/", token, "rewrap-test", true, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	reporter.now = func() time.Time { return time.Date(2026, 7, 11, 15, 0, 0, 0, time.UTC) }
	jobID := uuid.NewString()
	if err := reporter.Report(context.Background(), Summary{JobID: jobID}, "healthy"); err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer "+token || idempotency != "rewrap-"+jobID+"-healthy" {
		t.Fatalf("headers auth=%q idempotency=%q", authorization, idempotency)
	}
}

func TestHTTPReporterRejectsInsecureOrCredentialedEndpoints(t *testing.T) {
	for _, endpoint := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com/path", "https://example.com?token=x"} {
		if _, err := NewHTTPReporter(endpoint, strings.Repeat("x", 32), "rewrap-test", false, nil); err == nil {
			t.Fatalf("accepted endpoint %q", endpoint)
		}
	}
}

func TestHTTPReporterErrorNeverContainsToken(t *testing.T) {
	token := "rewrap-operations-token-that-is-long-enough"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, token, http.StatusBadGateway) }))
	defer server.Close()
	reporter, err := NewHTTPReporter(server.URL, token, "rewrap-test", true, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	err = reporter.Report(context.Background(), Summary{JobID: uuid.NewString()}, "failed")
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("unsafe reporter error: %v", err)
	}
}

func TestHTTPReporterNeverForwardsCredentialAcrossRedirect(t *testing.T) {
	received := false
	destination := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { received = true }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	reporter, err := NewHTTPReporter(source.URL, strings.Repeat("x", 32), "rewrap-test", true, source.Client())
	if err != nil {
		t.Fatal(err)
	}
	err = reporter.Report(context.Background(), Summary{JobID: uuid.NewString()}, "healthy")
	if err == nil || received {
		t.Fatalf("redirect followed=%v err=%v", received, err)
	}
}
