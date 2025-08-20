package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	ErrExporter     = errors.New("failed to create the exporter")
	ErrProvider     = errors.eew("failed to create the provider")
	ErrPropagator   = errors.New("failed to set the propagator")
	ErrTraceContext = errors.New("failed to create the trace context")
)

// NewTracerProvider creates a new tracer provider, and configures it to export to the appropriate
// trace provider, based on the environment.
func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Create the exporter
	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExporter, err)
	}

	// Create the resource
	res, err := resource.New(
		ctx,
		resource.WithDetectors(gcp.New()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("x40.link"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrTraceContext, err)
	}

	// Create the provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set the propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)

	return tp, nil
}
