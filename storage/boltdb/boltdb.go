package boltdb

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/andrewhowdencom/x40.link/storage"
	"go.etcd.io/bbolt"
)

var (
	ErrFailedToSetupDatabase = errors.New("failed to setup backing database")
	ErrFailedToTX            = errors.New("failed to complete database transaction")
	ErrDataCorrupt           = errors.New("data returned from the database corrupted")

	txBucketName = []byte("short-links")
)

// Option modifies the bolt options, allowing the user to set some property of the database.
type Option func(db *bbolt.Options) error

// BoltDB is an implementation of the link shortener that stores links in the
// boltdb storage engine by CoreOS (later, etcd-io):
//
// * https://github.com/etcd-io/bbolt
//
// Not something I've used a lot, so YMMV.
type BoltDB struct {
	db *bbolt.DB
}

// New creates a new BoltDB backed storage implementation.
func New(path string, opts ...Option) (*BoltDB, error) {
	// The initial options are derived from bbolt.DefaultOptions, with the timeout applied so it does not
	// lock forever.
	bopts := &bbolt.Options{
		Timeout:      1 * time.Second,
		NoGrowSync:   false,
		FreelistType: bbolt.FreelistArrayType,
	}

	// Allow users to override the underlying database options. While this exposes this, this implementation
	// is only a thin wrapper around BoltDB to handle the semantics of URLs.
	for _, o := range opts {
		if err := o(bopts); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToSetupDatabase, err)
		}
	}

	new, err := bbolt.Open(path, 0600, bopts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToSetupDatabase, err)
	}

	return &BoltDB{
		db: new,
	}, nil
}

// WithFileLockWait modifies the timeout that Unix based operating systems will wait for the file lock
// to be freed.
func WithFileLockWait(dur time.Duration) Option {
	return func(db *bbolt.Options) error {
		db.Timeout = dur

		return nil
	}
}

func (b *BoltDB) Get(in *url.URL) (*url.URL, error) {
	var u *url.URL

	if err := b.db.View(func(tx *bbolt.Tx) error {
		// If there's no bucket created, no put operations can have been run. Ergo, the key cannot exist.
		b := tx.Bucket(txBucketName)
		if b == nil {
			return storage.ErrNotFound
		}

		v := b.Get([]byte(in.String()))
		if v == nil {
			return storage.ErrNotFound
		}

		var err error
		u, err = url.Parse(string(v))
		if err != nil {
			return ErrDataCorrupt
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return u, nil
}

func (b *BoltDB) Put(f *url.URL, t *url.URL) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(txBucketName)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToTX, err)
		}

		if err := b.Put([]byte(f.String()), []byte(t.String())); err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToTX, err)
		}

		return nil
	})
}
