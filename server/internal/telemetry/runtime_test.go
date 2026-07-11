package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestLoadRuntimeConfigIsDisabledByDefaultAndValidatesExporter(t *testing.T) {
	disabled, err := LoadRuntimeConfig(func(string) string { return "" })
	if err != nil || disabled.Enabled {
		t.Fatalf("disabled=%+v err=%v", disabled, err)
	}

	values := map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT": "http://otel-collector:4318",
		"OTEL_EXPORTER_OTLP_INSECURE": "true",
		"OTEL_SERVICE_NAME":           "tenancit-e2e",
	}
	configured, err := LoadRuntimeConfig(func(key string) string { return values[key] })
	if err != nil || !configured.Enabled || configured.ServiceName != "tenancit-e2e" {
		t.Fatalf("configured=%+v err=%v", configured, err)
	}

	values["OTEL_EXPORTER_OTLP_ENDPOINT"] = "https://user:secret@collector.internal:4318?token=leak"
	if _, err := LoadRuntimeConfig(func(key string) string { return values[key] }); err == nil {
		t.Fatal("exporter endpoint accepted credentials and query")
	}
}

func TestRuntimeExportsTraceAndMetricToOTLPHTTPCollector(t *testing.T) {
	var traces atomic.Int32
	var metrics atomic.Int32
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/traces":
			traces.Add(1)
		case "/v1/metrics":
			metrics.Add(1)
		default:
			t.Errorf("unexpected OTLP path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()
	previousTracer := otel.GetTracerProvider()
	previousMeter := otel.GetMeterProvider()
	defer otel.SetTracerProvider(previousTracer)
	defer otel.SetMeterProvider(previousMeter)

	shutdown, err := SetupRuntime(context.Background(), RuntimeConfig{
		Enabled: true, Endpoint: collector.URL, Insecure: true, ServiceName: "tenancit-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	middleware, err := NewDefaultHTTPMiddleware()
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(shutdownContext); err != nil {
		t.Fatal(err)
	}
	if traces.Load() == 0 || metrics.Load() == 0 {
		t.Fatalf("collector received traces=%d metrics=%d", traces.Load(), metrics.Load())
	}
}
