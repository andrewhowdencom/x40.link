package storage

import (
	"context"
	"errors"
	"net/url"
)

// Err* are common errors that the storage implementations will return.
var (
	ErrNotFound           = errors.New("input url not found")
	ErrStorageSetupFailed = errors.New("storage setup failed")
	ErrReadOnlyStorage    = errors.New("storage is read only")
	ErrFailed             = errors.New("storage implementation failed")
	ErrCorrupt            = errors.New("the data returned by the storage is invalid")
	ErrUnauthorized       = errors.New("you are not the owner of this record")
)

// CtxKey is a type designed to allow delimiting key/value pairs
type CtxKey string

const (
	// CtxKeyAgent is the context address at which the owner will be found, if there is one.
	CtxKeyAgent CtxKey = "agent"
)

// Authenticator is an extension to the storage interface that allows verifying whether a given user is the canonical
// owner of a given URL.
//
// Expects to be supplied with a context with the agent stored at the appropriate key. Returns false in the case the
// user doesn't own the request, or the request cannot be authenticated (e.g. agent is missing)
type Authenticator interface {
	Owns(ctx context.Context, u *url.URL) bool
}

// Storer is the interface that retrieves links supplied to it. Methods are named after the RESTful HTTP
// verbs, as the meanings are semantically similar.
type Storer interface {
	// Get a URL, given a supplied shortlink.
	Get(ctx context.Context, url *url.URL) (*url.URL, error)

	// Store a map between a shortlink and the destination.
	Put(ctx context.Context, from *url.URL, to *url.URL) error
}
