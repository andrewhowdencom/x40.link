// Package server implements the HTTP server that will respond to the requests for URLs, sending the
// user to the appropriate location (or rejecting the response)
package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

// Option is a function type that modifies the behavior of the server
type Option func(*server) error

// Err* are sentinel errors
var (
	ErrFailedToApplyOption = errors.New("failed to apply option")
	ErrFailedToStart       = errors.New("failed to start server")
)

type server struct {
	*http.Server
	Trace *trace.TracerProvider
}

// New creates a server instance, configured appropriately
func New(opts ...Option) (*server, error) {
	srv := &server{
		Server: &http.Server{},
	}

	// Default options
	opts = append([]Option{
		WithListenAddress("localhost:80"),
		WithMiddleware(middleware.Recoverer),
		WithMiddleware(Error),
	}, opts...)

	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToApplyOption, err)
		}
	}

	// Now that we've configured the handler, lets wrap it in the final tracer
	if srv.Trace != nil {
		srv.Server.Handler = otelhttp.NewHandler(srv.Handler, "http", otelhttp.WithTracerProvider(srv.Trace))
	}

	return srv, nil
}

func (s *server) ListenAndServe() error {
	return s.Server.ListenAndServe()
}

// WithMiddleware appends middleware to the default handler
func WithMiddleware(m func(next http.Handler) http.Handler) Option {
	return func(srv *server) error {
		if srv.Handler == nil {
			srv.Handler = chi.NewRouter()
		}

		mux, ok := srv.Handler.(*chi.Mux)
		if !ok {
			return errors.New("cannot apply middleware to non-chi handler")
		}
		mux.Use(m)

		return nil
	}
}

// WithListenAddress indicates the server should start on the specific address
func WithListenAddress(addr string) Option {
	return func(s *server) error {
		s.Addr = addr

		return nil
	}
}

// WithStorage allows starting the service with a specific storage engine.
func WithStorage(str storage.Storer) Option {
	return func(srv *server) error {
		if srv.Handler == nil {
			srv.Handler = chi.NewRouter()
		}

		mux, ok := srv.Handler.(*chi.Mux)
		if !ok {
			return errors.New("cannot apply middleware to non-chi handler")
		}

		sh := &strHandler{
			str: str,
		}

		mux.Get("/*", sh.Redirect)

		return nil
	}
}

// WithH2C allows piping the connection to a HTTP/2 server, which will hijack the request to use the HTTP/2 protocol
// but over the initially supplied connection.
func WithH2C() Option {
	return func(srv *server) error {
		if srv.Handler == nil {
			srv.Handler = chi.NewRouter()
		}

		mux, ok := srv.Handler.(*chi.Mux)
		if !ok {
			return errors.New("cannot apply middleware to non-chi handler")
		}

		mux.Use(Intercept(IsH2C, h2c.NewHandler(
			mux,

			// The relevant HTTP/2 server to upgrade and hanadle connections on.
			&http2.Server{},
		)))

		return nil
	}
}

// WithGRPC enables GRPC to be served over the
func WithGRPC(host string, server *grpc.Server) Option {
	return func(srv *server) error {
		if srv.Handler == nil {
			srv.Handler = chi.NewRouter()
		}

		mux, ok := srv.Handler.(*chi.Mux)
		if !ok {
			return errors.New("cannot apply middleware to non-chi handler")
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

func WithTracerProvider(p *trace.TracerProvider) Option {
	return func(s *server) error {
		s.Trace = p
		return nil
	}
}
