package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type probeStub struct {
	name     string
	critical bool
	err      error
}

func TestHTTPProbeTreatsOnlySuccessfulResponsesAsHealthy(t *testing.T) {
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthy" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer receiver.Close()

	if err := NewHTTPProbe("oidc", receiver.URL+"/healthy", false, receiver.Client()).Check(context.Background()); err != nil {
		t.Fatalf("healthy probe: %v", err)
	}
	if err := NewHTTPProbe("oidc", receiver.URL+"/failed", false, receiver.Client()).Check(context.Background()); err == nil {
		t.Fatal("failed probe was reported healthy")
	}
}

func (p probeStub) Name() string                { return p.name }
func (p probeStub) Critical() bool              { return p.critical }
func (p probeStub) Check(context.Context) error { return p.err }

func TestEvaluateReadinessFailsOnlyForUnavailableCriticalDependencies(t *testing.T) {
	report := EvaluateReadiness(context.Background(), []Probe{
		probeStub{name: "postgres", critical: true},
		probeStub{name: "valkey", critical: true, err: errors.New("redis://secret@internal:6379 unavailable")},
		probeStub{name: "oidc", critical: false, err: errors.New("https://issuer.internal failed")},
	}, func() time.Time { return time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC) })

	if report.Status != StatusUnavailable || report.Components[0].Status != StatusHealthy {
		t.Fatalf("report=%+v", report)
	}
	if report.Components[1].Status != StatusUnavailable || report.Components[2].Status != StatusDegraded {
		t.Fatalf("components=%+v", report.Components)
	}
	for _, component := range report.Components {
		if component.Name == "" || component.LatencyMS < 0 {
			t.Fatalf("invalid component=%+v", component)
		}
	}
}

func TestReadinessJSONContainsNoProbeErrorOrConfiguration(t *testing.T) {
	report := EvaluateReadiness(context.Background(), []Probe{
		probeStub{name: "postgres", critical: true, err: errors.New("postgres://admin:token@db.private/tenancit")},
	}, time.Now)

	encoded, err := report.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"admin", "token", "db.private", "postgres://"} {
		if containsString(string(encoded), forbidden) {
			t.Fatalf("readiness leaked %q: %s", forbidden, encoded)
		}
	}
}

func containsString(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
