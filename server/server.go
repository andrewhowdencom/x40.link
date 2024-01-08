// Package server implements the HTTP server that will respond to the requests for URLs, sending the
// user to the appropriate location (or rejecting the response)
package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/andrewhowdencom/x40.link/api"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/net/http2"
)

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
		mux := srv.Handler.(*chi.Mux)
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

// WithStorage allows starting the service with a specific storage engine.
func WithStorage(str storage.Storer) Option {
	return func(srv *http.Server) error {
		mux := srv.Handler.(*chi.Mux)

		sh := &strHandler{
			str: str,
		}

		mux.Get("/{slug}", sh.Redirect)

		return nil
	}
}

// WithGRPCGateway configures an interceptor to offload requests to the GRPC Gateway mux. Must be used before
// any option that creates a route (e.g. WithStorage)
func WithGRPCGateway() Option {
	return func(srv *http.Server) error {
		mux := srv.Handler.(*chi.Mux)

		mux.Use(Intercept(GRPCGateway{api.NewGRPCGatewayMux()}))

		return nil
	}
}

// WithH2C allows piping the connection to a HTTP/2 server, which will hijack the request to use the HTTP/2 protocol
// but over the initially supplied connection.
func WithH2C(h2 *http2.Server) Option {
	return func(srv *http.Server) error {
		mux := srv.Handler.(*chi.Mux)
		mux.Use(Intercept(H2C{h2}))

		return nil
	}
}

// WithGRPC enables GRPC to be served over the
func WithGRPC() Option {
	return func(srv *http.Server) error {
		mux := srv.Handler.(*chi.Mux)

		mux.Use(Intercept(GRPC{api.NewGRPCMux()}))

		return nil
	}
}
