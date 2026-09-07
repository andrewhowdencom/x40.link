package boltdb

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newTempDB returns a BoltDB instance backed by a file in t.TempDir(),
// along with a cleanup function that closes the database and removes
// the file.
func newTempDB(t *testing.T) (*BoltDB, func()) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "x40.db")
	db, err := New(path)
	require.NoError(t, err)

	cleanup := func() {
		_ = db.db.Close()
		_ = os.RemoveAll(filepath.Dir(path))
	}

	return db, cleanup
}

// withSpanRecorder swaps in an OTel TracerProvider backed by an
// in-memory SpanRecorder and returns it. The previous provider is
// restored when the test finishes.
func withSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	return rec
}

func TestBoltDB_GetEmitsStorageLookupSpan(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel TracerProvider.

	rec := withSpanRecorder(t)
	db, cleanup := newTempDB(t)
	defer cleanup()

	// Seed an entry.
	require.NoError(t, db.Put(
		context.Background(),
		&url.URL{Scheme: "https", Host: "example.local", Path: "/foo"},
		&url.URL{Scheme: "https", Host: "example.local", Path: "/bar"},
	))

	// Replace the global span with one named "test" so we can wrap
	// the call in a parent span (and ensure the child span carries the
	// proper parent relationship).
	ctx, span := otel.Tracer("test").Start(context.Background(), "test-parent")

	_, _ = db.Get(ctx, &url.URL{Scheme: "https", Host: "example.local", Path: "/foo"})

	span.End()

	spans := rec.Ended()
	require.NotEmpty(t, spans, "expected at least one span to be recorded")

	var found bool
	for _, s := range spans {
		if s.Name() == "storage.lookup" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a span named storage.lookup in %v", spanNames(spans))
}

func TestBoltDB_PutEmitsStorageWriteSpan(t *testing.T) {
	// No t.Parallel: this test mutates the global OTel TracerProvider.

	rec := withSpanRecorder(t)
	db, cleanup := newTempDB(t)
	defer cleanup()

	ctx, span := otel.Tracer("test").Start(context.Background(), "test-parent")

	require.NoError(t, db.Put(
		ctx,
		&url.URL{Scheme: "https", Host: "example.local", Path: "/foo"},
		&url.URL{Scheme: "https", Host: "example.local", Path: "/bar"},
	))

	span.End()

	spans := rec.Ended()
	require.NotEmpty(t, spans, "expected at least one span to be recorded")

	var found bool
	for _, s := range spans {
		if s.Name() == "storage.write" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a span named storage.write in %v", spanNames(spans))
}

func spanNames(spans []sdktrace.ReadOnlySpan) []string {
	out := make([]string, len(spans))
	for i, s := range spans {
		out[i] = s.Name()
	}
	return out
}