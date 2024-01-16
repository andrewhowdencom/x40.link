package server

import (
	"net/http"

	"github.com/andrewhowdencom/x40.link/server/message"
)

// MatcherFunc is a function that can be used by the interceptor to match requests.
type MatcherFunc func(*http.Request) bool

// AllOf combines multiple matchers into a single matcher func
func AllOf(matchers ...MatcherFunc) MatcherFunc {
	return func(r *http.Request) bool {
		for _, m := range matchers {
			if !m(r) {
				return false
			}
		}

		return true
	}
}

// IsHost matches whether or not a request matches a specific host
func IsHost(host string) MatcherFunc {
	return func(r *http.Request) bool {
		return r.Host == host
	}
}

// IsGRPC offloads requests to the gRPC mux. Note: This does not use a bunch of GRPC features; that's fine.
//
// See
// - https://github.com/philips/grpc-gateway-example/blob/master/cmd/serve.go#L51-L61
// - https://ahmet.im/blog/grpc-http-mux-go/
func IsGRPC(r *http.Request) bool {

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

// IsH2C does the detection of the initial message. The message looks like:
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
func IsH2C(r *http.Request) bool {
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

// Intercept is a type of middleware that offloads messages that are destined for the "default" handler and instead
// redirects them to some other handler.
//
// This allows using more complex matching logic. See IsGRPCGateway for an example.
func Intercept(matches MatcherFunc, intercept http.Handler) func(next http.Handler) http.Handler {
	return func(standard http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if matches(r) {
				intercept.ServeHTTP(w, r)
				return
			}

			standard.ServeHTTP(w, r)
		})
	}
}
