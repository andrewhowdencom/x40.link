// Package server implements the HTTP server that will respond to the requests for URLs, sending the
// user to the appropriate location (or rejecting the response)
package server

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/gin-gonic/gin"
)

// Server is a wrapper around the gin engine, allowing us to configure it as
// we see fit.
type Server struct {
	engine *gin.Engine

	listen string
}

// Option is a function type that modifies the behavior of the server
type Option func(*Server) error

// Err* are sentinel errors
var (
	ErrFailedToApplyOption = errors.New("failed to apply option")
	ErrFailedToStart       = errors.New("failed to start server")
)

var defaultOptions = []Option{
	WithListenAddress("localhost:80"),
	WithGinMode(gin.ReleaseMode),
}

// New creates a server instance, configured appropriately
func New(opts ...Option) (*Server, error) {
	r := gin.New()
	srv := &Server{
		engine: r,
	}

	opts = append(defaultOptions, opts...)

	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToApplyOption, err)
		}
	}

	return srv, nil
}

// Start will start the server
//
// This will block the invoking goroutine indefinitely, unless an error occurs
func (s *Server) Start() error {
	if err := s.engine.Run(s.listen); err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToStart, err)
	}

	return nil
}

// WithListenAddress indicates the server should start on the specific address
func WithListenAddress(addr string) Option {
	return func(s *Server) error {
		s.listen = addr

		return nil
	}
}

// WithStorage allows starting the service with a specific storage engine
func WithStorage(str storage.Storer) Option {
	return func(srv *Server) error {
		sh := &strHandler{
			str: str,
		}

		srv.engine.NoRoute(sh.Redirect)

		return nil
	}
}

// WithGinMode sets the gin mode. Warning: This sets a global property for all gin instances.
func WithGinMode(m string) Option {
	return func(s *Server) error {
		gin.SetMode(m)

		return nil
	}
}
