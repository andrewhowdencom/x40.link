package api

import (
	"context"

	"github.com/andrewhowdencom/x40.link/telemetry"
	"github.com/google/wire"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTracerProvider creates a new tracer provider, and configures it to export to the appropriate
// trace provider, based on the environment.
func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	return telemetry.NewTracerProvider(ctx)
}

var TracerProviderSet = wire.NewSet(NewTracerProvider)
