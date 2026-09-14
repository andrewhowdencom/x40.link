package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// resetOtelGlobals captures the OTel globals and restores them after the
// test, since initTelemetry mutates them.
func resetOtelGlobals(t *testing.T) {
	t.Helper()
	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
	})
}

func resetViper(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { viper.Reset() })
}

func TestInitTelemetry_DisabledReturnsNoopShutdown(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	viper.Set(cfg.OTELEnabled.Path, false)

	shutdown, err := initTelemetry(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown must be a no-op and safe to invoke.
	require.NoError(t, shutdown(context.Background()))
}

func TestInitTelemetry_DisabledShutdownIdempotent(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	viper.Set(cfg.OTELEnabled.Path, false)

	shutdown, err := initTelemetry(context.Background())
	require.NoError(t, err)

	require.NoError(t, shutdown(context.Background()))
	require.NoError(t, shutdown(context.Background()))
}

func TestInitTelemetry_EnabledDoesNotPanic(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	endpoint := "localhost:14317"
	viper.Set(cfg.OTELEnabled.Path, true)
	viper.Set(cfg.OTELExporterEndpoint.Path, endpoint)
	viper.Set(cfg.OTELServiceName.Path, "x40.link-test")
	// The probe runs against a non-existent port — disable it so the
	// test doesn't sit on a 3-second timeout that's unrelated to
	// what we're trying to verify (that construction succeeds).
	viper.Set(cfg.OTELProbeEndpoint.Path, false)

	shutdown, err := initTelemetry(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// The exporter cannot reach localhost:14317, so a synchronous flush
	// will time out — we don't assert NoError on shutdown. We only
	// require that the call returns within a bounded time.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
