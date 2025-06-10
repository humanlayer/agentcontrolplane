package otel

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	DefaultOTLPEndpoint = "otel-collector-opentelemetry-collector:4318"
	ServiceName         = "acp-controller"
)

// InitTracer initializes the OpenTelemetry tracer provider with an OTLP HTTP exporter.
// If the OTEL collector is not available, it returns a no-op tracer provider.
func InitTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultOTLPEndpoint
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		// If OTEL collector is not available, use a no-op tracer provider
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(ServiceName),
			)),
		)
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(ServiceName),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

// InitMeter initializes the OpenTelemetry meter provider with an OTLP HTTP exporter.
// If the OTEL collector is not available, it returns a no-op meter provider.
func InitMeter(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultOTLPEndpoint
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		// If OTEL collector is not available, use a no-op meter provider
		mp := sdkmetric.NewMeterProvider()
		otel.SetMeterProvider(mp)
		return mp, nil
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	otel.SetMeterProvider(mp)
	return mp, nil
}
