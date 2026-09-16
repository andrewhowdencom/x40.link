// Package server implements the HTTP server that will respond to the requests for URLs, sending the
// user to the appropriate location (or rejecting the response)
package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/andrewhowdencom/x40.link/server/message"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	//nolint:staticcheck // SA1019: golang.org/x/net/http2/h2c is deprecated.
	// Migrate to http.Server.Protocols = []http.Protocol{"h2c"} (Go 1.24+)
	// once chi's h2c helper supports it; tracked as a follow-up.
	"golang.org/x/net/http2"
	//nolint:staticcheck // SA1019: golang.org/x/net/http2/h2c is deprecated.
	// Same as above.
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

// SpanNameRedirect is the OpenTelemetry span name emitted by the HTTP
// redirect handler. Defined here so it can be referenced by tests and
// dashboards without depending on the package internals.
const SpanNameRedirect = "redirect"

// WithOtel wraps the server's standard request chain in
// otelhttp.NewMiddleware so the HTTP redirect emits a span named
// SpanNameRedirect. gRPC requests are filtered out (otelgrpc handles
// them independently) and the chi `Intercept` middleware on the mux
// routes them away from the standard chain before otelhttp sees them.
func WithOtel() Option {
	return func(srv *http.Server) error {
		mux, err := serverMux(srv)
		if err != nil {
			return err
		}

		mux.Use(otelhttp.NewMiddleware(SpanNameRedirect,
			otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string {
				return SpanNameRedirect
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return r.Header.Get(message.HeaderContentType) != message.MIMEGRPC
			}),
		))

		return nil
	}
}

// Option is a function type that modifies the behavior of the server
type Option func(*http.Server) error

// Err* are sentinel errors
var (
	ErrFailedToApplyOption = errors.New("failed to apply option")
	ErrFailedToStart       = errors.New("failed to start server")
)

var defaultOptions = []Option{
	WithListenAddress("localhost:80"),
	WithMiddleware(middleware.Recoverer),
	WithMiddleware(Error),
}

// New creates a server instance, configured appropriately
func New(opts ...Option) (*http.Server, error) {
	srv := &http.Server{
		Handler: chi.NewRouter(),
	}

	opts = append(defaultOptions, opts...)

	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToApplyOption, err)
		}
	}

	return srv, nil
}

// WithMiddleware appends middleware to the default handler
func WithMiddleware(m func(next http.Handler) http.Handler) Option {
	return func(srv *http.Server) error {
		mux, err := serverMux(srv)
		if err != nil {
			return err
		}
		mux.Use(m)

		return nil
	}
}

// WithListenAddress indicates the server should start on the specific address
func WithListenAddress(addr string) Option {
	return func(s *http.Server) error {
		s.Addr = addr

		return nil
	}
}

// WithStorage allows starting the service with a specific storage
// engine. The supplied name is used as the "storage" label on the
// custom business counters emitted by the redirect handler.
func WithStorage(str storage.Storer, name string) Option {
	return func(srv *http.Server) error {
		mux, err := serverMux(srv)
		if err != nil {
			return err
		}

		sh := &instrumentedRedirect{
			inner:   &strHandler{str: str},
			storage: name,
		}

		mux.Get("/*", sh.Redirect)

		return nil
	}
}

// WithH2C wraps the configured handler with an HTTP/2 cleartext server.
//
// The H2C handler deliberately sits outside chi. Installing it as chi
// middleware causes the HTTP/2 connection context to contain chi's
// request-scoped routing context. Every multiplexed stream then shares
// that mutable context, producing routing races and spurious responses.
func WithH2C() Option {
	return func(srv *http.Server) error {
		mux, err := serverMux(srv)
		if err != nil {
			return err
		}

		srv.Handler = &h2cServerHandler{
			Handler: h2c.NewHandler( //nolint:staticcheck // SA1019: h2c.NewHandler is deprecated; migrate to http.Server.Protocols.
				mux,
				&http2.Server{}, //nolint:staticcheck // SA1019: h2c.Server is deprecated.
			),
			mux: mux,
		}

		return nil
	}
}

// h2cServerHandler keeps the connection-level H2C handler outside chi
// while retaining the mux for options applied later.
type h2cServerHandler struct {
	http.Handler
	mux *chi.Mux
}

func serverMux(srv *http.Server) (*chi.Mux, error) {
	switch handler := srv.Handler.(type) {
	case *chi.Mux:
		return handler, nil
	case *h2cServerHandler:
		return handler.mux, nil
	default:
		return nil, fmt.Errorf("server handler %T does not contain a chi mux", srv.Handler)
	}
}

// WithGRPC enables GRPC to be served over the
func WithGRPC(host string, server *grpc.Server) Option {
	return func(srv *http.Server) error {
		mux, err := serverMux(srv)
		if err != nil {
			return err
		}
		filters := []MatcherFunc{
			IsGRPC,
		}

		// Allow the GRPC Gateway to filter to specific hosts, if required.
		if host != "" {
			filters = append(filters, IsHost(host))
		}

		mux.Use(Intercept(AllOf(filters...), server))

		return nil
	}
}
