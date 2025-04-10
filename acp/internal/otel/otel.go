package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer initializes the OpenTelemetry tracer provider with an OTLP HTTP exporter.
func InitTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otel-collector-opentelemetry-collector:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("kubechain-controller"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

// InitMeter initializes the OpenTelemetry meter provider with an OTLP HTTP exporter.
func InitMeter(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("otel-collector-opentelemetry-collector:4318"),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	otel.SetMeterProvider(mp)
	return mp, nil
}
