package telemetry

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOperationalSignalsNormalizeUntrustedDimensions(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	defer otel.SetMeterProvider(previous)

	RecordSecurityDecision(context.Background(), "token=super-secret", "denied")
	RecordDependencyOperation(context.Background(), "postgres", "SELECT * FROM secrets", "error", 10*time.Millisecond)
	RecordWorkerCycle(context.Background(), "webhook", "success", 3, 20*time.Millisecond)
	RecordRewrapBatch(context.Background(), "success", 2, 30*time.Millisecond)
	RecordRewrapRows(context.Background(), 1, 2, "success", 2)
	RecordRewrapRemaining(context.Background(), 1, 2)
	RecordRewrapCompletion(context.Background(), "success", 2, 0, time.Second)
	RecordRewrapFailure(context.Background(), "authentication")

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	encoded := formatMetricAttributes(metrics)
	for _, expected := range []string{"tenancit.security.decisions", "tenancit.dependency.operations", "tenancit.worker.cycles", "tenancit.rewrap.batches", "tenancit.rewrap.rows.by_version", "tenancit.rewrap.rows.remaining.by_version", "tenancit.rewrap.failures", "other"} {
		if !strings.Contains(encoded, expected) {
			t.Fatalf("metrics missing %q: %s", expected, encoded)
		}
	}
	for _, forbidden := range []string{"super-secret", "SELECT", "secrets"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("metrics leaked %q: %s", forbidden, encoded)
		}
	}
}
