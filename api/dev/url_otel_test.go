package dev

import (
	"context"
	"net/url"
	"testing"

	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
)

// withSpanRecorder sets the global TracerProvider to one backed by an
// in-memory SpanRecorder and returns the recorder so the test can inspect
// it. The previous provider is restored when the test finishes.
func withSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(trace.NewTracerProvider(trace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	return rec
}

// startSpanInCtx creates a context with an open span named name, plus a
// defer-able End callback. Used by the rename tests to simulate the
// otelgrpc interceptor's parent span.
func startSpanInCtx(ctx context.Context, t *testing.T, name string) (context.Context, func()) {
	t.Helper()

	ctx, span := otel.Tracer("test").Start(ctx, name)
	return ctx, func() { span.End() }
}

func TestInstrumentedURL_GetRenamesSpanToResolveLink(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel TracerProvider.

	rec := withSpanRecorder(t)

	// Build an inner URL with a fixture that returns successfully.
	str := test.New()
	require.NoError(t, str.Put(
		context.Background(),
		&url.URL{Scheme: "https", Host: "example.local", Path: "/foo"},
		&url.URL{Scheme: "https", Host: "example.local", Path: "/bar"},
	))

	inner := &URL{Storer: str}
	wrapped := InstrumentURL(inner, "hashmap")

	// Simulate the parent span that otelgrpc would create on the way
	// into the handler. The wrapper must rename it.
	ctx, end := startSpanInCtx(context.Background(), t, "/x40.dev.url.ManageURLs/Get")

	resp, err := wrapped.Get(ctx, &gendev.GetRequest{Url: "https://example.local/foo"})
	require.NoError(t, err)
	require.NotNil(t, resp)

	end() // close the parent span so the recorder sees it.

	spans := rec.Ended()
	require.Len(t, spans, 1, "expected exactly one ended span")
	assert.Equal(t, "resolve_link", spans[0].Name(),
		"the span must be renamed from the gRPC method to the business operation")
}

func TestInstrumentedURL_NewRenamesSpanToCreateLink(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel TracerProvider.

	rec := withSpanRecorder(t)

	str := test.New()
	inner := &URL{
		Storer:   str,
		Enricher: func(_, _ *url.URL) error { return nil },
	}
	wrapped := InstrumentURL(inner, "hashmap")

	ctx, end := startSpanInCtx(context.Background(), t, "/x40.dev.url.ManageURLs/New")

	resp, err := wrapped.New(ctx, &gendev.NewRequest{
		On:     &gendev.RedirectOn{Host: "example.local", Path: "/foo"},
		SendTo: "https://example.local/bar",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	end()

	spans := rec.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "create_link", spans[0].Name())
}

// TestInstrumentedURL_Compiles ensures the wrapper implements the
// gendev.ManageURLsServer interface. The compile-time check is also
// expressed as `var _ gendev.ManageURLsServer = (*instrumentedURL)(nil)`
// in url_otel.go, but a runtime assertion keeps the test self-documenting.
func TestInstrumentedURL_Compiles(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel TracerProvider.

	var _ gendev.ManageURLsServer = (*instrumentedURL)(nil)

	// And: a no-op trace provider is acceptable; renaming a no-op span
	// must not panic.
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(noop.NewTracerProvider())
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	inner := &URL{Storer: test.New()}
	wrapped := InstrumentURL(inner, "hashmap")

	ctx, end := startSpanInCtx(context.Background(), t, "/x40.dev.url.ManageURLs/Get")
	_, _ = wrapped.Get(ctx, &gendev.GetRequest{Url: "https://nope.local"})
	end()
	// Pass criterion: no panic.
}
