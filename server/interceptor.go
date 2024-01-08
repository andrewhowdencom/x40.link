package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/andrewhowdencom/x40.link/server/message"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
)

// bufConn wraps a net.Conn, but reads drain the bufio.Reader first.
type bufConn struct {
	net.Conn
	*bufio.Reader
}

func (c *bufConn) Read(p []byte) (int, error) {
	if c.Reader == nil {
		return c.Conn.Read(p)
	}
	n := c.Reader.Buffered()
	if n == 0 {
		c.Reader = nil
		return c.Conn.Read(p)
	}
	if n < len(p) {
		p = p[:n]
	}
	return c.Reader.Read(p)
}

// Err* are sentinel errors
var (
	ErrUnableToHijackConnection = errors.New("unable to hijack connection")
	ErrReadingClientPreface     = errors.New("error reading h2c client preface")
)

// Interceptor is a superset type of HTTP Handler that includes matching logic.
type Interceptor interface {
	http.Handler

	Match(r *http.Request) bool
}

// GRPCGateway offloads requests destined to the GRPCGateway to the embedded handler
type GRPCGateway struct {
	*runtime.ServeMux
}

// Match indicates that a request should be intercepted and sent to the gRPC Gateway
func (gw GRPCGateway) Match(r *http.Request) bool {
	// There is nothing else on this server that is expecting a JSON response. Given this, forward everything
	// JSON related to the handler.
	return r.Header.Get(message.HeaderAccept) == message.MIMEApplicationJSON
}

// GRPC offloads requests destined for GRPC directly.
type GRPC struct {
	*grpc.Server
}

// Match offloads requests to the gRPC mux. Note: This does not use a bunch of GRPC features; that's fine.
//
// See
// - https://github.com/philips/grpc-gateway-example/blob/master/cmd/serve.go#L51-L61
// - https://ahmet.im/blog/grpc-http-mux-go/
func (g GRPC) Match(r *http.Request) bool {

	// GRPC has its own mime type.
	if r.Header.Get(message.HeaderContentType) != message.MIMEGRPC {
		return false
	}

	// GRPC only works over HTTP/2
	if r.ProtoMajor != 2 {
		return false
	}

	return true
}

// H2C introspects the request and, if it appears to be a HTTP request with prior knowledge, passes it to the HTTP/2
// handler. Only a partial re-implementation of H2C middleware, as the Upgrade path was deprecated in RFC9113
//
// See
// 1. https://pkg.go.dev/golang.org/x/net/http2/h2c
// 2. https://github.com/golang/go/discussions/60746
type H2C struct {
	*http2.Server
}

// Match does the detection of the initial message. The message looks like:
//
//	PRI * HTTP/2.0
//
//	SM
//
//
//	<Byte Data>
//
// (At least, as reproduced by $ curl --http2-prior-knowledge). See:
// 1. https://www.rfc-editor.org/rfc/rfc7540#section-4.1
func (h2c H2C) Match(r *http.Request) bool {
	if r.Method != "PRI" {
		return false
	}

	if len(r.Header) != 0 {
		return false
	}

	if r.URL.Path != "*" {
		return false
	}

	if r.Proto != "HTTP/2.0" {
		return false
	}

	return true
}

// ServeHTTP initializes the HTTP/2 connection.
// See http2/h2c:initH2CWithPriorKnowledge for details.
func (h2c H2C) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hijacker, ok := w.(http.Hijacker)

	if !ok {
		WithError(req, fmt.Errorf("%w: %s", ErrUnableToHijackConnection, "does not meet hijacker interface"))
		return
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		WithError(req, fmt.Errorf("%w: %s", ErrUnableToHijackConnection, err.Error()))
		return
	}

	// Read the prior knowledge message into a buffer, and validate it.
	expected := "SM\r\n\r\n"
	buf := make([]byte, len(expected))
	n, err := io.ReadFull(rw, buf)

	if err != nil {
		WithError(req, fmt.Errorf("%w: %s", ErrReadingClientPreface, err))
		return
	}

	// If the message does not match the "indicate upgrade", cancel the connection.
	if string(buf[:n]) != expected {
		WithError(req, fmt.Errorf("%w: %s", ErrReadingClientPreface, "preface message not correct"))

		// The error is ignored as we are already in an error handler.
		_ = conn.Close()
		return
	}

	// We do not mind whether the flush works.
	_ = rw.Flush()
	if rw.Reader.Buffered() != 0 {
		conn = &bufConn{conn, rw.Reader}
	}

	defer conn.Close() //nolint:errcheck

	// Fetch initial server out of the existing request, and use it as base configuration to serve the same request
	// over HTTP/2
	srv := req.Context().Value(http.ServerContextKey).(*http.Server)
	h2c.ServeConn(conn, &http2.ServeConnOpts{
		Context:          req.Context(),
		BaseConfig:       srv,
		Handler:          srv.Handler,
		SawClientPreface: true,
	})
}

// Intercept is a type of middleware that offloads messages that are destined for the "default" handler and instead
// redirects them to some other handler.
//
// This allows using more complex matching logic. See IsGRPCGateway for an example.
func Intercept(interceptor Interceptor) func(next http.Handler) http.Handler {
	return func(standard http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if interceptor.Match(r) {
				interceptor.ServeHTTP(w, r)
				return
			}

			standard.ServeHTTP(w, r)
		})
	}
}
