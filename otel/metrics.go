// Package otel initialises OpenTelemetry TracerProvider and MeterProvider from
// application configuration, returning a single shutdown closure that flushes
// both signals to OTLP/gRPC.
package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Surface* constants identify the surface (HTTP / gRPC) that initiated
// the metric. They are exported so dashboards can filter on the value
// without depending on internal magic strings.
const (
	SurfaceHTTP = "http"
	SurfaceGRPC = "grpc"
)

// Op* constants identify the storage operation that failed. They are
// exported for the same reason as Surface*.
const (
	OpLookup = "lookup"
	OpWrite  = "write"
)

// Custom business metric names. Exported so dashboards and tests can
// reference them without depending on string literals embedded in
// multiple call sites.
const (
	MetricLinksCreated  = "x40.link.links.created"
	MetricLinksResolved = "x40.link.links.resolved"
	MetricLinksNotFound = "x40.link.links.not_found"
	MetricStorageErrors = "x40.link.storage.errors"
)

// meter is the package's OTel meter. The SDK's Meter lookup caches
// instruments internally, so calling Int64Counter on every Record*
// invocation is cheap. Looking up the counter on each call (rather
// than caching at package init) also means tests that swap the global
// meter provider between tests see the correct counter without needing
// a reset hook.
func meter() metric.Meter {
	return otel.Meter("x40.link")
}

func attrOf(kv ...attribute.KeyValue) metric.MeasurementOption {
	return metric.WithAttributes(kv...)
}

func strAttr(k, v string) attribute.KeyValue {
	return attribute.String(k, v)
}

// RecordCreated increments the "links created" counter for the given
// storage backend.
func RecordCreated(storage string) {
	c, _ := meter().Int64Counter(MetricLinksCreated)
	c.Add(context.Background(), 1,
		attrOf(strAttr("storage", storage)))
}

// RecordResolved increments the "links resolved" counter. surface
// identifies the originating protocol (SurfaceHTTP or SurfaceGRPC).
func RecordResolved(storage, surface string) {
	c, _ := meter().Int64Counter(MetricLinksResolved)
	c.Add(context.Background(), 1,
		attrOf(
			strAttr("storage", storage),
			strAttr("surface", surface),
		))
}

// RecordNotFound increments the "links not found" counter.
func RecordNotFound(storage, surface string) {
	c, _ := meter().Int64Counter(MetricLinksNotFound)
	c.Add(context.Background(), 1,
		attrOf(
			strAttr("storage", storage),
			strAttr("surface", surface),
		))
}

// RecordStorageError increments the "storage errors" counter. op
// identifies the failing operation (OpLookup or OpWrite).
func RecordStorageError(storage, op string) {
	c, _ := meter().Int64Counter(MetricStorageErrors)
	c.Add(context.Background(), 1,
		attrOf(
			strAttr("storage", storage),
			strAttr("op", op),
		))
}
