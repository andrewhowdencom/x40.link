package server

import (
	"context"
	"net/http"

	"github.com/andrewhowdencom/x40.link/server/message"
	"schneider.vip/problem"
)

type key string

// Ctx* are context value keys
const (
	CtxErrors key = "error"
)

// Problem* are common types of problems
var (
	ProblemUnknown = problem.New(
		problem.Status(http.StatusInternalServerError),
		problem.Title("An unexpected error has occurred"),
		problem.Detail("The server has encountered an unexpected error. There's nothing, as a user, you can do. Please try again later"))
)

// WithError adds the error to the current request. Middleware later picks it out, and writes
// out the status.
//
// https://cs.opensource.google/go/go/+/refs/tags/go1.21.6:src/net/http/server.go;l=2141-2150
// https://github.com/go-chi/render/blob/14f1cb3d5c2969d6e462632a205eacb6421eb4dc/responder.go#L25-L26
func WithError(r *http.Request, err error) {
	*r = *r.WithContext(context.WithValue(r.Context(), CtxErrors, err))
}

// Error cancels the request processing
func Error(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Run all subsequent handlers. We only want the failures.
		next.ServeHTTP(w, r)

		v := r.Context().Value(CtxErrors)

		// If there is nothing here, the middleware has nothing to do.
		if v == nil {
			return
		}

		var p *problem.Problem

		// If there is already a problem on the request, simply use this.
		if np, ok := v.(*problem.Problem); ok {
			p = np
		} else if _, ok := v.(error); ok {
			p = ProblemUnknown
		} else {
			panic("a non-error type added as error context")
		}

		switch r.Header.Get(message.Accept) {
		// Both application/xml and text/xml are sometimes used.
		case message.MIMETextXML:
			fallthrough
		case message.MIMEApplicationXML:
			// Errors are ignored here. In future, they should be logged against a trace (or similar)
			_, _ = p.WriteXMLTo(w)
		// The application/json is the "default" version, but it is called out here for clarity in
		// the code. There is no supported "text" version.
		case message.MIMEApplicationJSON:
			fallthrough
		default:

			// Errors are ignored here. In future, they should be logged against a trace (or similar)
			_, _ = p.WriteTo(w)
		}
	})
}
