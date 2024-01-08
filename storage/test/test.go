// Package test provides a storage implementation designed explicitly for testing
package test

import (
	"net/url"

	"github.com/andrewhowdencom/x40.link/storage"
)

// Option modifies the behavior of New() in specific ways that make it valuable for the test.
type Option func(t *ts)

// ts is test storage
type ts struct {
	r map[string]*url.URL

	// error will modify the test structure to return an error for all operations.
	err error
}

// New generates the test storage implementation
func New(opts ...Option) *ts {
	n := &ts{}

	for _, o := range opts {
		o(n)
	}

	return n
}

// WithError modifies the storage implementation such that any operation executed against it will return
// an error.
func WithError(err error) Option {
	return func(t *ts) {
		t.err = err
	}
}

// see storage.Storer
func (ts *ts) Get(u *url.URL) (*url.URL, error) {
	if ts.err != nil {
		return nil, ts.err
	}

	if v, ok := ts.r[u.String()]; ok {
		return v, nil
	}

	return nil, storage.ErrNotFound
}

// see storage.Storer
func (ts *ts) Put(f *url.URL, t *url.URL) error {
	if ts.err != nil {
		return ts.err
	}

	ts.r[f.String()] = t
	return nil
}
