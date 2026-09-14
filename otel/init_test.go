// Package otel initialises OpenTelemetry TracerProvider and MeterProvider from
// application configuration, returning a single shutdown closure that flushes
// both signals to OTLP/gRPC.
package otel

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/version"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// otelEnv holds the OTel-related viper keys for a single test. Each test
// constructs one, applies it, and discards it via cleanup. Tests that
// touch viper or the otel globals must NOT use t.Parallel — they share
// global state.
type otelEnv struct {
	enabled  *bool
	endpoint *string
	name     *string
	attrs    *string
	insecure *bool
	probe    *bool
}

func (e otelEnv) apply(t *testing.T) {
	t.Helper()
	if e.enabled != nil {
		viper.Set(cfg.OTELEnabled.Path, *e.enabled)
	}
	if e.endpoint != nil {
		viper.Set(cfg.OTELExporterEndpoint.Path, *e.endpoint)
	}
	if e.name != nil {
		viper.Set(cfg.OTELServiceName.Path, *e.name)
	}
	if e.attrs != nil {
		viper.Set(cfg.OTELResourceAttributes.Path, *e.attrs)
	}
	if e.insecure != nil {
		viper.Set(cfg.OTELExporterInsecure.Path, *e.insecure)
	}
	if e.probe != nil {
		viper.Set(cfg.OTELProbeEndpoint.Path, *e.probe)
	}
}

func disabled() *bool { v := false; return &v }
func enabled() *bool  { v := true; return &v }

// otelDisabledWithProbeOff sets up an OTel environment that's disabled —
// all probes and credential code paths are skipped because Init returns
// early when OTEL is disabled.
func otelDisabled() otelEnv {
	return otelEnv{enabled: disabled()}
}

// otelLocalhostInsecure sets up the standard sidecar/local-collector
// endpoint (localhost:4317, insecure) with the startup probe disabled.
// The tests that use this target a non-existent port; they care about
// the construction / wiring, not network reachability.
func otelLocalhostInsecure() otelEnv {
	endpoint := "localhost:4317"
	name := "x40.link-test"
	insecure := true
	probe := false
	return otelEnv{
		enabled:  enabled(),
		endpoint: &endpoint,
		name:     &name,
		insecure: &insecure,
		probe:    &probe,
	}
}

// resetOtelGlobals captures the OTel globals and restores them in cleanup.
// Required because Init mutates the globals; tests must not leak providers.
func resetOtelGlobals(t *testing.T) {
	t.Helper()

	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()

	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
	})
}

// resetViper wipes any value bound to the OTel config paths so each test
// starts from a clean slate.
func resetViper(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { viper.Reset() })
}

func TestInit_DisabledIsNoop(t *testing.T) {
	// No t.Parallel — shares viper.
	resetViper(t)
	resetOtelGlobals(t)

	otelDisabled().apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown must be safe and a no-op (returns nil) when OTel is disabled.
	require.NoError(t, shutdown(context.Background()))
}

func TestInit_EnabledProducesShutdown(t *testing.T) {
	// No t.Parallel — shares viper and otel globals.
	resetViper(t)
	resetOtelGlobals(t)

	otelLocalhostInsecure().apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// The exporter cannot reach localhost:4317, so a synchronous flush
	// will time out — we don't assert NoError on shutdown. We only
	// require that the call returns within a bounded time.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}

func TestInit_DisabledShutdownIsIdempotent(t *testing.T) {
	// No t.Parallel — shares viper.
	resetViper(t)

	otelDisabled().apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)

	require.NoError(t, shutdown(context.Background()))
	require.NoError(t, shutdown(context.Background()))
}

// TestInit_InsecureFlagControlsTLS verifies that cfg.OTELExporterInsecure
// toggles whether the OTLP exporters are built without TLS. Both paths
// must construct a non-nil shutdown without panicking; the dial itself
// never succeeds against the bogus endpoint, but the test exercises
// the option wiring.
func TestInit_InsecureFlagControlsTLS(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	endpoint := "localhost:4317"
	cases := []struct {
		name     string
		insecure bool
	}{
		{"insecure", true},
		{"tls", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			insecure := tc.insecure
			probe := false
			otelEnv{
				enabled:  enabled(),
				endpoint: &endpoint,
				insecure: &insecure,
				probe:    &probe,
			}.apply(t)

			shutdown, err := Init(context.Background())
			require.NoError(t, err)
			require.NotNil(t, shutdown)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = shutdown(ctx)
		})
	}
}

// TestInit_StartupProbeFails verifies that the startup probe surfaces a
// clear error when the configured OTLP endpoint is not listening.
func TestInit_StartupProbeFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow probe test in short mode")
	}

	resetViper(t)
	resetOtelGlobals(t)

	// A port we are confident is not in use. Probing is fast: the
	// connect attempt returns connection-refused within milliseconds.
	endpoint := "127.0.0.1:1"
	insecure := true
	otelEnv{
		enabled:  enabled(),
		endpoint: &endpoint,
		insecure: &insecure,
	}.apply(t)

	start := time.Now()
	shutdown, err := Init(context.Background())
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Contains(t, err.Error(), "startup probe")
	// shutdown is noopShutdown (a non-nil no-op closure) when Init
	// fails partway through; the contract is that the caller must
	// invoke it even on error, to flush whatever providers built
	// before the failure.
	require.NotNil(t, shutdown)
	// Generous bound — the probe is 3s, and the OTel SDK exporters
	// may attempt a connection on construction (taking up to ~5s
	// each in some environments). The real assertion is "did it fail
	// fast enough to be useful as a startup gate?"; 15s is the
	// upper bound for that.
	assert.Less(t, elapsed, 15*time.Second,
		"probe must fail in bounded time (got %v)", elapsed)
}

// TestInit_StartupProbeSucceeds verifies the probe path on a real
// listening socket. The caller is responsible for closing the socket
// within the test lifetime; Init returns a shutdown that has been
// registered with the OTel SDK.
func TestInit_StartupProbeSucceeds(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	// Listen on an ephemeral port on loopback.
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	port := listener.Addr().(*net.TCPAddr).Port
	endpoint := "127.0.0.1:" + itoa(port)
	insecure := true
	otelEnv{
		enabled:  enabled(),
		endpoint: &endpoint,
		insecure: &insecure,
	}.apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)
}

func itoa(p int) string {
	// Minimal itoa to avoid a strconv import just for the test.
	if p == 0 {
		return "0"
	}
	digits := []byte{}
	for p > 0 {
		digits = append([]byte{byte('0' + p%10)}, digits...)
		p /= 10
	}
	return string(digits)
}

// TestParseResourceAttributes — held over from the original spec;
// covers the comma-separated OTEL_RESOURCE_ATTRIBUTES parser.
func TestParseResourceAttributes(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, parseResourceAttributes(""))
	})

	t.Run("single", func(t *testing.T) {
		t.Parallel()
		got := parseResourceAttributes("a=1")
		require.Len(t, got, 1)
		assert.Equal(t, "1", got[0].Value.AsString())
		assert.Equal(t, attribute.Key("a"), got[0].Key)
	})

	t.Run("multiple with whitespace", func(t *testing.T) {
		t.Parallel()
		got := parseResourceAttributes("a=1, b=2 , c=3")
		require.Len(t, got, 3)
	})

	t.Run("malformed entries skipped", func(t *testing.T) {
		t.Parallel()
		got := parseResourceAttributes("good=1,bad,also-good=2")
		require.Len(t, got, 2)
	})
}

func TestResolveOTLPEndpoint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		raw        string
		insecure   bool
		wantTarget string
		wantHost   string
		wantURL    bool
		wantPlain  bool
	}{
		{
			name:       "flag host and port",
			raw:        "telemetry.googleapis.com:443",
			wantTarget: "telemetry.googleapis.com:443",
			wantHost:   "telemetry.googleapis.com",
		},
		{
			name:       "standard HTTPS URL",
			raw:        "https://telemetry.googleapis.com",
			wantTarget: "telemetry.googleapis.com:443",
			wantHost:   "telemetry.googleapis.com",
			wantURL:    true,
		},
		{
			name:       "standard HTTP URL",
			raw:        "http://localhost:4317",
			wantTarget: "localhost:4317",
			wantHost:   "localhost",
			wantURL:    true,
			wantPlain:  true,
		},
		{
			name:       "explicit plaintext flag",
			raw:        "localhost:4317",
			insecure:   true,
			wantTarget: "localhost:4317",
			wantHost:   "localhost",
			wantPlain:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveOTLPEndpoint(tc.raw, tc.insecure)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTarget, got.target)
			assert.Equal(t, tc.wantHost, got.hostname)
			assert.Equal(t, tc.wantURL, got.fromURL)
			assert.Equal(t, tc.wantPlain, got.insecure)
		})
	}
}

func TestGCPProjectID(t *testing.T) {
	t.Parallel()

	detected := resource.NewSchemaless(semconv.CloudAccountID("example-project"))
	assert.Equal(t, "example-project", gcpProjectID(detected))
}

// TestVersion_DefaultValue pins the default for the build-time
// injected version variable.
func TestVersion_DefaultValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", version.Version)
}
