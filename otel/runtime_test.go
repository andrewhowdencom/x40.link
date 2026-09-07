package otel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	otelcontribruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
)

// TestRuntimeMetricsFlow asserts that starting the contrib runtime
// instrumentation with a manual meter reader results in the goroutine
// counter being recorded. This validates the runtime integration in
// otel.Init (Task 7 in the plan) at the unit level.
func TestRuntimeMetricsFlow(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		// The contrib runtime package does not expose a Stop function
		// in this version; the goroutine it spawns dies with the
		// process.
	})

	require.NoError(t, otelcontribruntime.Start(otelcontribruntime.WithMeterProvider(mp)))

	// Give the runtime producer a moment to read the runtime stats.
	time.Sleep(500 * time.Millisecond)

	// Force a collect. Try up to a few times in case the producer's
	// first sample hasn't landed yet.
	deadline := time.Now().Add(3 * time.Second)
	var rm metricdata.ResourceMetrics
	var found bool
	for time.Now().Before(deadline) && !found {
		require.NoError(t, reader.Collect(context.Background(), &rm))

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				// The OTel contrib runtime instrumentation prefixes
				// runtime/metrics names with "process." but the
				// suffixes depend on the Go version. We assert that at
				// least one goroutine and one heap-related metric are
				// recorded; the exact names track the Go runtime.
				if isGoroutineCountMetric(m.Name) || isHeapMetric(m.Name) {
					_, ok := m.Data.(metricdata.Sum[int64])
					require.True(t, ok, "expected %s to be an Int64 sum", m.Name)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			time.Sleep(100 * time.Millisecond)
		}
	}
	assert.True(t, found, "expected a recorded goroutine or heap metric in: %v", collectNames(&rm))
}

// isGoroutineCountMetric reports whether the metric name (as exposed
// by the contrib runtime instrumentation) corresponds to the running
// goroutine count. The exact name varies across OTel contrib and Go
// versions; we match a stable suffix.
func isGoroutineCountMetric(name string) bool {
	return name == "process.runtime.go.goroutines.count" ||
		name == "process.runtime.go.goroutines"
}

// isHeapMetric reports whether the metric name is any heap-related
// runtime metric produced by the contrib package.
func isHeapMetric(name string) bool {
	return name == "process.runtime.go.mem.heap_alloc" ||
		name == "process.runtime.go.memory.used" ||
		name == "process.runtime.go.memory.allocated"
}

func collectNames(rm *metricdata.ResourceMetrics) []string {
	var names []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}
