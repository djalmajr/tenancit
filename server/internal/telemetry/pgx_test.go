package telemetry

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPGXTracerRecordsOutcomeWithoutSQLOrArguments(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	defer otel.SetMeterProvider(previous)
	tracer := NewPGXTracer()

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL:  "SELECT * FROM secrets WHERE token=$1",
		Args: []any{"raw-secret-token"},
	})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	encoded := formatMetricAttributes(metrics)
	if !strings.Contains(encoded, "tenancit.dependency.operations") || !strings.Contains(encoded, "query") {
		t.Fatalf("missing query metric: %s", encoded)
	}
	for _, forbidden := range []string{"SELECT", "secrets", "raw-secret-token", "token=$1"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("metric leaked %q: %s", forbidden, encoded)
		}
	}
}
