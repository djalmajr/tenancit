package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHTTPMiddlewareExportsOnlyRouteTemplateAndBoundedAttributes(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	middleware, err := NewHTTPMiddleware(meterProvider.Meter("test"), tracerProvider.Tracer("test"))
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/v1/admin/items/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/items/item-secret?token=query-secret", nil)
	request.Header.Set("Authorization", "Bearer header-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans=%d", len(spans))
	}
	spanText := spans[0].Name()
	for _, attribute := range spans[0].Attributes() {
		spanText += " " + string(attribute.Key) + "=" + attribute.Value.Emit()
	}
	if !strings.Contains(spanText, "/v1/admin/items/{id}") {
		t.Fatalf("trace missing route template: %s", spanText)
	}
	for _, forbidden := range []string{"item-secret", "query-secret", "header-secret", "cookie-secret", "token="} {
		if strings.Contains(spanText, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, spanText)
		}
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	metricText := formatMetricAttributes(metrics)
	if !strings.Contains(metricText, "/v1/admin/items/{id}") || strings.Contains(metricText, "secret") {
		t.Fatalf("metric attributes=%s", metricText)
	}
}

func formatMetricAttributes(metrics metricdata.ResourceMetrics) string {
	var output strings.Builder
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			output.WriteString(metric.Name)
			switch data := metric.Data.(type) {
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					for _, attribute := range point.Attributes.ToSlice() {
						output.WriteString(string(attribute.Key) + "=" + attribute.Value.Emit())
					}
				}
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					for _, attribute := range point.Attributes.ToSlice() {
						output.WriteString(string(attribute.Key) + "=" + attribute.Value.Emit())
					}
				}
			}
		}
	}
	return output.String()
}
