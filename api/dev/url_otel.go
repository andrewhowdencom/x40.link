package dev

import (
	"context"

	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"go.opentelemetry.io/otel/trace"
)

// Span name constants used by instrumentedURL. Exported as constants so
// tests, dashboards, and documentation can reference them by name
// without depending on the wrapper's internal structure.
const (
	SpanNameResolveLink = "resolve_link"
	SpanNameCreateLink  = "create_link"
)

// instrumentedURL wraps a gendev.ManageURLsServer implementation and
// renames the auto-instrumented OpenTelemetry span (typically named
// "/x40.dev.url.ManageURLs/Get" or "/x40.dev.url.ManageURLs/New") to the
// corresponding business operation name defined in AGENTS.md.
//
// The wrapper does not start a new span of its own; it renames the
// parent span started by the otelgrpc interceptor. This avoids the
// double-span pattern that AGENTS.md warns against ("Do not duplicate
// traces for business operations versus HTTP requests").
type instrumentedURL struct {
	gendev.UnimplementedManageURLsServer

	inner gendev.ManageURLsServer
}

// Compile-time check that *instrumentedURL satisfies the interface.
var _ gendev.ManageURLsServer = (*instrumentedURL)(nil)

// InstrumentURL wraps the given gendev.ManageURLsServer so spans
// emitted by the otelgrpc interceptor are renamed to business operation
// names. This is the public seam the api package uses to register the
// service with the gRPC server.
func InstrumentURL(inner gendev.ManageURLsServer) gendev.ManageURLsServer {
	return &instrumentedURL{inner: inner}
}

// Get renames the inbound span to SpanNameResolveLink and delegates to
// the wrapped implementation.
func (i *instrumentedURL) Get(ctx context.Context, req *gendev.GetRequest) (*gendev.Response, error) {
	trace.SpanFromContext(ctx).SetName(SpanNameResolveLink)
	return i.inner.Get(ctx, req)
}

// New renames the inbound span to SpanNameCreateLink and delegates to
// the wrapped implementation.
func (i *instrumentedURL) New(ctx context.Context, req *gendev.NewRequest) (*gendev.Response, error) {
	trace.SpanFromContext(ctx).SetName(SpanNameCreateLink)
	return i.inner.New(ctx, req)
}
