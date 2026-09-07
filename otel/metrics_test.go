package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// withManualMeter swaps in a MeterProvider backed by a manual reader so
// the test can inspect emitted metrics. The previous global is restored
// when the test finishes.
func withManualMeter(t *testing.T) *metric.ManualReader {
	t.Helper()

	prev := otel.GetMeterProvider()
	reader := metric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(prev) })

	return reader
}

// findInt64Counter returns the first Int64-sum datapoint matching the
// named counter, or nil if no points have been recorded yet.
func findInt64Counter(t *testing.T, reader *metric.ManualReader, name string) *metricdata.DataPoint[int64] {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			intSum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "expected %s to be an Int64 counter", name)

			for i := range intSum.DataPoints {
				return &intSum.DataPoints[i]
			}
		}
	}
	return nil
}

func attr(t *testing.T, pt *metricdata.DataPoint[int64], key string) string {
	t.Helper()
	v, ok := pt.Attributes.Value(attribute.Key(key))
	require.True(t, ok, "expected attribute %q on counter", key)
	return v.AsString()
}

// Note: these tests do NOT use t.Parallel because they mutate the
// global OTel MeterProvider. Running them in parallel would race
// over which provider is "current" at the moment RecordX is called.

func TestRecordCreated(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel MeterProvider.

	reader := withManualMeter(t)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	RecordCreated("boltdb")

	pt := findInt64Counter(t, reader, "x40.link.links.created")
	require.NotNil(t, pt, "expected a recorded x40.link.links.created datapoint")
	assert.Equal(t, int64(1), pt.Value)
	assert.Equal(t, "boltdb", attr(t, pt, "storage"))
}

func TestRecordResolved(t *testing.T) {
	reader := withManualMeter(t)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	RecordResolved("hashmap", SurfaceGRPC)

	pt := findInt64Counter(t, reader, "x40.link.links.resolved")
	require.NotNil(t, pt)
	assert.Equal(t, int64(1), pt.Value)
	assert.Equal(t, "hashmap", attr(t, pt, "storage"))
	assert.Equal(t, SurfaceGRPC, attr(t, pt, "surface"))
}

func TestRecordNotFound(t *testing.T) {
	reader := withManualMeter(t)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	RecordNotFound("boltdb", SurfaceHTTP)

	pt := findInt64Counter(t, reader, "x40.link.links.not_found")
	require.NotNil(t, pt)
	assert.Equal(t, int64(1), pt.Value)
	assert.Equal(t, "boltdb", attr(t, pt, "storage"))
	assert.Equal(t, SurfaceHTTP, attr(t, pt, "surface"))
}

func TestRecordStorageError(t *testing.T) {
	reader := withManualMeter(t)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	RecordStorageError("firestore", OpWrite)

	pt := findInt64Counter(t, reader, "x40.link.storage.errors")
	require.NotNil(t, pt)
	assert.Equal(t, int64(1), pt.Value)
	assert.Equal(t, "firestore", attr(t, pt, "storage"))
	assert.Equal(t, OpWrite, attr(t, pt, "op"))
}

// TestLink4MetricsFire ensures all four custom business metrics are
// registered as instruments with the expected names and can be invoked
// without panicking. This is a smoke test that catches registration
// regressions.
func TestLink4MetricsFire(t *testing.T) {
	reader := withManualMeter(t)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	RecordCreated("a")
	RecordResolved("a", SurfaceGRPC)
	RecordNotFound("a", SurfaceGRPC)
	RecordStorageError("a", OpLookup)

	for _, name := range []string{
		"x40.link.links.created",
		"x40.link.links.resolved",
		"x40.link.links.not_found",
		"x40.link.storage.errors",
	} {
		pt := findInt64Counter(t, reader, name)
		require.NotNil(t, pt, "expected a recorded datapoint for %s", name)
		assert.Equal(t, int64(1), pt.Value)
	}
}
