package dev

import (
	"context"
	"errors"

	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/otel"
	"github.com/andrewhowdencom/x40.link/storage"
	"go.opentelemetry.io/otel/trace"
)

// Span name constants used by instrumentedURL. Exported as constants so
// tests, dashboards, and documentation can reference them by name
// without depending on the wrapper's internal structure.
const (
	SpanNameResolveLink = "resolve_link"
	SpanNameCreateLink  = "create_link"
	SpanNameListLinks   = "list_links"
)

// instrumentedURL wraps a gendev.ManageURLsServer implementation and:
//
//   - renames the OpenTelemetry span (started by the otelgrpc
//     interceptor) from the gRPC method name to the corresponding
//     business operation name defined in AGENTS.md;
//   - emits the package's custom business counters on the resolved,
//     not-found, and created paths.
//
// The wrapper does not start a new span of its own; it renames the
// parent span started by the otelgrpc interceptor. This avoids the
// double-span pattern that AGENTS.md warns against ("Do not duplicate
// traces for business operations versus HTTP requests").
type instrumentedURL struct {
	gendev.UnimplementedManageURLsServer

	inner   gendev.ManageURLsServer
	storage string
}

// Compile-time check that *instrumentedURL satisfies the interface.
var _ gendev.ManageURLsServer = (*instrumentedURL)(nil)

// InstrumentURL wraps the given gendev.ManageURLsServer so spans
// emitted by the otelgrpc interceptor are renamed to business operation
// names, and so the package's custom business counters are emitted with
// the supplied storage label (e.g. "boltdb", "hashmap").
//
// This is the public seam the api package uses to register the service
// with the gRPC server.
func InstrumentURL(inner gendev.ManageURLsServer, storage string) gendev.ManageURLsServer {
	return &instrumentedURL{inner: inner, storage: storage}
}

// Get renames the inbound span to SpanNameResolveLink, delegates to
// the wrapped implementation, and emits the appropriate custom
// business counter based on the outcome.
func (i *instrumentedURL) Get(ctx context.Context, req *gendev.GetRequest) (*gendev.Response, error) {
	trace.SpanFromContext(ctx).SetName(SpanNameResolveLink)

	resp, err := i.inner.Get(ctx, req)
	switch {
	case err == nil:
		otel.RecordResolved(i.storage, otel.SurfaceGRPC)
	case errors.Is(err, storage.ErrNotFound):
		otel.RecordNotFound(i.storage, otel.SurfaceGRPC)
	}
	return resp, err
}

// New renames the inbound span to SpanNameCreateLink, delegates to
// the wrapped implementation, and emits the links.created counter on
// success.
func (i *instrumentedURL) New(ctx context.Context, req *gendev.NewRequest) (*gendev.Response, error) {
	trace.SpanFromContext(ctx).SetName(SpanNameCreateLink)

	resp, err := i.inner.New(ctx, req)
	if err == nil {
		otel.RecordCreated(i.storage)
	}
	return resp, err
}

// List renames the inbound gRPC span to the business operation.
func (i *instrumentedURL) List(ctx context.Context, req *gendev.ListRequest) (*gendev.ListResponse, error) {
	trace.SpanFromContext(ctx).SetName(SpanNameListLinks)
	return i.inner.List(ctx, req)
}
