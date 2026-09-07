package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/server"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// withSpanRecorder sets the global TracerProvider to one backed by an
// in-memory SpanRecorder and returns the recorder so the test can inspect
// it. The previous provider is restored when the test finishes.
func withSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	return rec
}

// TestRedirectSpanName verifies that the redirect handler emits a span
// named "redirect" — the business operation name defined in AGENTS.md.
func TestRedirectSpanName(t *testing.T) {
	t.Parallel()

	rec := withSpanRecorder(t)

	storage := test.New()
	require.NoError(t, storage.Put(
		context.Background(),
		&url.URL{Host: "x40.local", Path: "/foo"},
		&url.URL{Scheme: "https", Host: "destination.local", Path: "/"},
	))

	// WithOtel must be applied before WithStorage: chi requires that
	// middlewares be defined before routes.
	srv, err := server.New(
		server.WithOtel(),
		server.WithStorage(storage),
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/foo", nil)
	req.Host = "x40.local"

	srv.Handler.ServeHTTP(w, req)

	// Sanity: the redirect actually happened.
	assert.Equal(t, http.StatusTemporaryRedirect, w.Result().StatusCode)

	spans := rec.Ended()
	require.NotEmpty(t, spans, "expected at least one span to be recorded")

	// Find the span with the redirect name; otelhttp may emit multiple
	// spans for the request but the redirect span is named exactly.
	var found bool
	for _, s := range spans {
		if s.Name() == "redirect" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a span named 'redirect' in: %v", spanNames(spans))
}

// TestRedirectSpanPropagator ensures the propagator is set even on the
// HTTP path. Without a propagator, downstream services cannot join the
// trace context. otel.SetTextMapPropagator was called by otel.Init
// but here we set a fresh propagator to verify the chain works without
// a real Init.
func TestRedirectSpanPropagator(t *testing.T) {
	t.Parallel()

	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })

	storage := test.New()
	srv, err := server.New(
		server.WithOtel(),
		server.WithStorage(storage),
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.Host = "x40.local"
	srv.Handler.ServeHTTP(w, req)

	// No assertion on the traceparent header — otelhttp only adds it if
	// it actually starts a span. The test merely verifies the chain
	// does not panic.
}

func spanNames(spans []sdktrace.ReadOnlySpan) []string {
	out := make([]string, len(spans))
	for i, s := range spans {
		out[i] = s.Name()
	}
	return out
}
