package storage

import (
	"errors"
	"net/url"
)

// Err* are common errors that the storage implementations will return.
var (
	ErrNotFound           = errors.New("input url not found")
	ErrStorageSetupFailed = errors.New("storage setup failed")
	ErrReadOnlyStorage    = errors.New("storage is read only")
)

// Storer is the interface that retrieves links supplied to it. Methods are named after the RESTful HTTP
// verbs, as the meanings are semantically similar.
type Storer interface {
	// Get a URL, given a supplied shortlink.
	Get(url *url.URL) (*url.URL, error)

	// Store a map between a shortlink and the destination.
	Put(from *url.URL, to *url.URL) error
}
