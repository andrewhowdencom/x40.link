package storage

import (
	"errors"
	"net/url"
)

// Err* are common errors that the storage implementations will return.
var (
	ErrNotFound           = errors.New("input url not found")
	ErrStorageSetupFailed = errors.New("storage setup failed")
)

// Storer is the interface that retrieves links supplied to it.
type Storer interface {
	Get(url *url.URL) (*url.URL, error)
}
