package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type RuntimeConfig struct {
	Enabled     bool
	Endpoint    string
	Insecure    bool
	ServiceName string
}

func LoadRuntimeConfig(getenv func(string) string) (RuntimeConfig, error) {
	endpoint := strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return RuntimeConfig{ServiceName: "tenancit"}, nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return RuntimeConfig{}, errors.New("OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	insecure := strings.EqualFold(strings.TrimSpace(getenv("OTEL_EXPORTER_OTLP_INSECURE")), "true")
	if parsed.Scheme == "http" && !insecure {
		return RuntimeConfig{}, errors.New("HTTP OTLP endpoint requires OTEL_EXPORTER_OTLP_INSECURE=true")
	}
	serviceName := strings.TrimSpace(getenv("OTEL_SERVICE_NAME"))
	if serviceName == "" {
		serviceName = "tenancit"
	}
	if !operationalName.MatchString(serviceName) {
		return RuntimeConfig{}, errors.New("OTEL_SERVICE_NAME contains unsupported characters")
	}
	return RuntimeConfig{Enabled: true, Endpoint: parsed.String(), Insecure: insecure, ServiceName: serviceName}, nil
}

var operationalName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func SetupRuntime(ctx context.Context, config RuntimeConfig) (func(context.Context) error, error) {
	if !config.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	traceOptions := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(config.Endpoint)}
	metricOptions := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(config.Endpoint)}
	if config.Insecure {
		traceOptions = append(traceOptions, otlptracehttp.WithInsecure())
		metricOptions = append(metricOptions, otlpmetrichttp.WithInsecure())
	}
	traceExporter, err := otlptracehttp.New(ctx, traceOptions...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx, metricOptions...)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}
	serviceResource, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(config.ServiceName),
	))
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		_ = metricExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(serviceResource),
	)
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(10*time.Second))),
		metric.WithResource(serviceResource),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return func(shutdownContext context.Context) error {
		return errors.Join(tracerProvider.Shutdown(shutdownContext), meterProvider.Shutdown(shutdownContext))
	}, nil
}
