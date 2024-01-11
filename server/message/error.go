package message

import (
	"context"
	"net/http"
)

type key string

// Ctx* are context value keys
const (
	CtxErrors key = "error"
)

// WithError adds the error to the current request. Middleware later picks it out, and writes
// out the status.
//
// https://cs.opensource.google/go/go/+/refs/tags/go1.21.6:src/net/http/server.go;l=2141-2150
// https://github.com/go-chi/render/blob/14f1cb3d5c2969d6e462632a205eacb6421eb4dc/responder.go#L25-L26
func WithError(r *http.Request, err error) {
	*r = *r.WithContext(context.WithValue(r.Context(), CtxErrors, err))
}
