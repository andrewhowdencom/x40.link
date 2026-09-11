// Package otel initialises OpenTelemetry TracerProvider and MeterProvider from
// application configuration, returning a single shutdown closure that flushes
// both signals to OTLP/gRPC.
package otel

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/version"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

// attrByKey returns the value of the named attribute on a resource, or
// ("", false) if the attribute is not present.
func attrByKey(attrs []attribute.KeyValue, key attribute.Key) (attribute.Value, bool) {
	for _, kv := range attrs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

// disabled / enabled shortcuts.
func disabled() *bool { v := false; return &v }
func enabled() *bool  { v := true; return &v }

func TestInit_DisabledIsNoop(t *testing.T) {
	// No t.Parallel — shares viper.
	resetViper(t)
	resetOtelGlobals(t)

	otelEnv{enabled: disabled()}.apply(t)

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

	endpoint := "localhost:14317"
	name := "x40.link-test"
	otelEnv{
		enabled:  enabled(),
		endpoint: &endpoint,
		name:     &name,
	}.apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// The exporter can't reach localhost:14317, so a synchronous flush
	// will time out — we don't assert NoError on shutdown. We only
	// require that the call returns within a bounded time and doesn't
	// deadlock.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}

func TestInit_DisabledShutdownIsIdempotent(t *testing.T) {
	// No t.Parallel — shares viper.
	resetViper(t)

	otelEnv{enabled: disabled()}.apply(t)

	shutdown, err := Init(context.Background())
	require.NoError(t, err)

	require.NoError(t, shutdown(context.Background()))
	require.NoError(t, shutdown(context.Background()))
}

// TestInit_InsecureFlagControlsTLS verifies that the
// cfg.OTELExporterInsecure flag toggles whether the OTLP exporters
// are built without TLS. Both paths must construct a non-nil
// shutdown without panicking; the dial itself never succeeds
// against the bogus endpoint, but the test exercises the option
// wiring.
func TestInit_InsecureFlagControlsTLS(t *testing.T) {
	resetViper(t)
	resetOtelGlobals(t)

	endpoint := "localhost:14317"
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
			viper.Set(cfg.OTELExporterEndpoint.Path, endpoint)
			viper.Set(cfg.OTELExporterInsecure.Path, tc.insecure)

			shutdown, err := Init(context.Background())
			require.NoError(t, err)
			require.NotNil(t, shutdown)

			// Cancelled context makes shutdown return promptly without
			// needing the (unreachable) exporter to flush.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = shutdown(ctx)
		})
	}
}

func TestBuildResource_ServiceNameFromCfg(t *testing.T) {
	// No t.Parallel — calls resource.New which reads env vars.
	got, err := buildResource(context.Background(), resourceAttrs{
		serviceName: "x40.link-test",
		version:     "",
	})
	require.NoError(t, err)

	val, ok := attrByKey(got.Attributes(), "service.name")
	require.True(t, ok, "service.name must be set")
	assert.Equal(t, "x40.link-test", val.AsString())
}

func TestBuildResource_CloudRunFromEnv(t *testing.T) {
	// No t.Parallel — sets env vars.
	t.Setenv("K_REVISION", "rev-1")
	t.Setenv("K_SERVICE", "svc-1")

	got, err := buildResource(context.Background(), resourceAttrs{
		serviceName: "x40.link",
		revision:    os.Getenv("K_REVISION"),
		service:     os.Getenv("K_SERVICE"),
		version:     "v1.2.3",
	})
	require.NoError(t, err)

	rev, ok := attrByKey(got.Attributes(), "cloud.run.revision")
	require.True(t, ok, "cloud.run.revision must be set when K_REVISION is present")
	assert.Equal(t, "rev-1", rev.AsString())

	svc, ok := attrByKey(got.Attributes(), "cloud.run.service")
	require.True(t, ok)
	assert.Equal(t, "svc-1", svc.AsString())
}

func TestBuildResource_VersionAttribute(t *testing.T) {
	got, err := buildResource(context.Background(), resourceAttrs{
		serviceName: "x40.link",
		version:     "9.9.9",
	})
	require.NoError(t, err)

	v, ok := attrByKey(got.Attributes(), "service.version")
	require.True(t, ok)
	assert.Equal(t, "9.9.9", v.AsString())
}

func TestBuildResource_UserSuppliedAttributes(t *testing.T) {
	got, err := buildResource(context.Background(), resourceAttrs{
		serviceName:        "x40.link",
		resourceAttributes: "deployment.environment=ci,team=platform",
	})
	require.NoError(t, err)

	env, ok := attrByKey(got.Attributes(), "deployment.environment")
	require.True(t, ok)
	assert.Equal(t, "ci", env.AsString())

	team, ok := attrByKey(got.Attributes(), "team")
	require.True(t, ok)
	assert.Equal(t, "platform", team.AsString())
}

func TestBuildResource_MalformedAttributeSkipped(t *testing.T) {
	got, err := buildResource(context.Background(), resourceAttrs{
		serviceName:        "x40.link",
		resourceAttributes: "no-equals-sign",
	})
	require.NoError(t, err)

	// The malformed key must not appear on the resource.
	_, ok := attrByKey(got.Attributes(), "no-equals-sign")
	assert.False(t, ok)
}

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

// TestVersion_DefaultValue pins the default for the build-time injected
// version variable.
func TestVersion_DefaultValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", version.Version)
}
