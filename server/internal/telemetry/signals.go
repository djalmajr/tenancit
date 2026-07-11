package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	securityBoundaries = allowedValues("api_key_auth", "api_scope", "admin_identity", "admin_permission", "csrf", "rate_limit", "operations_report")
	signalOutcomes     = allowedValues("success", "allowed", "denied", "limited", "error", "unavailable")
	dependencyNames    = allowedValues("postgres", "valkey", "oidc", "audit", "usage", "outbox")
	dependencyOps      = allowedValues("query", "insert", "flush", "claim", "deliver", "retain", "ping")
	workerNames        = allowedValues("usage", "usage_retention", "webhook", "webhook_retention")
)

func RecordSecurityDecision(ctx context.Context, boundary, outcome string) {
	boundary = boundedValue(boundary, securityBoundaries)
	outcome = boundedValue(outcome, signalOutcomes)
	attributes := []attribute.KeyValue{
		attribute.String("tenancit.security.boundary", boundary),
		attribute.String("tenancit.outcome", outcome),
	}
	counter, _ := otel.Meter("github.com/djalmajr/tenancit/server/security").Int64Counter(
		"tenancit.security.decisions", metric.WithUnit("{decision}"),
	)
	counter.Add(ctx, 1, metric.WithAttributes(attributes...))
	trace.SpanFromContext(ctx).AddEvent("security.decision", trace.WithAttributes(attributes...))
}

func RecordDependencyOperation(ctx context.Context, component, operation, outcome string, duration time.Duration) {
	component = boundedValue(component, dependencyNames)
	operation = boundedValue(operation, dependencyOps)
	outcome = boundedValue(outcome, signalOutcomes)
	attributes := []attribute.KeyValue{
		attribute.String("tenancit.component", component),
		attribute.String("tenancit.operation", operation),
		attribute.String("tenancit.outcome", outcome),
	}
	meter := otel.Meter("github.com/djalmajr/tenancit/server/dependencies")
	counter, _ := meter.Int64Counter("tenancit.dependency.operations", metric.WithUnit("{operation}"))
	histogram, _ := meter.Float64Histogram("tenancit.dependency.duration", metric.WithUnit("s"))
	counter.Add(ctx, 1, metric.WithAttributes(attributes...))
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attributes...))
	trace.SpanFromContext(ctx).AddEvent("dependency.operation", trace.WithAttributes(attributes...))
}

func RecordWorkerCycle(ctx context.Context, worker, outcome string, items int, duration time.Duration) {
	worker = boundedValue(worker, workerNames)
	outcome = boundedValue(outcome, signalOutcomes)
	attributes := []attribute.KeyValue{
		attribute.String("tenancit.worker", worker),
		attribute.String("tenancit.outcome", outcome),
	}
	meter := otel.Meter("github.com/djalmajr/tenancit/server/workers")
	cycles, _ := meter.Int64Counter("tenancit.worker.cycles", metric.WithUnit("{cycle}"))
	processed, _ := meter.Int64Counter("tenancit.worker.items", metric.WithUnit("{item}"))
	histogram, _ := meter.Float64Histogram("tenancit.worker.duration", metric.WithUnit("s"))
	cycles.Add(ctx, 1, metric.WithAttributes(attributes...))
	processed.Add(ctx, int64(max(0, items)), metric.WithAttributes(attributes...))
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attributes...))
}

func allowedValues(values ...string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[value] = struct{}{}
	}
	return allowed
}

func boundedValue(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return "other"
}
