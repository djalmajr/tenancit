package telemetry

import (
	"fmt"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func NewHTTPMiddleware(meter metric.Meter, tracer trace.Tracer) (func(http.Handler) http.Handler, error) {
	requests, err := meter.Int64Counter("tenancit.http.server.requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request counter: %w", err)
	}
	duration, err := meter.Float64Histogram("tenancit.http.server.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create HTTP duration histogram: %w", err)
	}
	active, err := meter.Int64UpDownCounter("tenancit.http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create HTTP active request counter: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(ctx, "HTTP "+r.Method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attribute.String("http.request.method", r.Method)),
			)
			defer span.End()
			active.Add(ctx, 1, metric.WithAttributes(attribute.String("http.request.method", r.Method)))
			defer active.Add(ctx, -1, metric.WithAttributes(attribute.String("http.request.method", r.Method)))

			captured := httpsnoop.CaptureMetrics(next, w, r.WithContext(ctx))
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			attributes := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", captured.Code),
			}
			requests.Add(ctx, 1, metric.WithAttributes(attributes...))
			duration.Record(ctx, captured.Duration.Seconds(), metric.WithAttributes(attributes...))
			span.SetName(r.Method + " " + route)
			span.SetAttributes(attributes...)
			if captured.Code >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, "server_error")
			}
		})
	}, nil
}

func NewDefaultHTTPMiddleware() (func(http.Handler) http.Handler, error) {
	return NewHTTPMiddleware(otel.Meter("github.com/djalmajr/tenancit/server/http"), otel.Tracer("github.com/djalmajr/tenancit/server/http"))
}
