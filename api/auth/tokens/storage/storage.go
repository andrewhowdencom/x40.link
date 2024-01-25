// Package storage provides storage for the cachable token interface.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Err* are sentinel errors.
var (
	ErrNotFound       = errors.New("token not found")
	ErrStorageFailure = errors.New("storage failure")
)

// Storage is the interface that stores and retrieves bytes from disk. It is ... similar to the io.ReadWriter,
// except the Read/Write methods make assumptions that the underlying data should be truncated before
// new data is written in, and that users always want the full set of bytes.
type Storage interface {
	Read() ([]byte, error)
	Write([]byte) error
}

// File provides simple file-backed storage for the token interface.
type File struct {
	Path string
}

// Read implements storage.Read
func (f *File) Read() ([]byte, error) {
	fh, err := os.Open(f.Path)

	// We treat a file not being found as a token that is not created yet, as this is
	// the case that may happen.
	if err != nil && os.IsNotExist(err) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	b, err := io.ReadAll(fh)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	return b, nil
}

// Write implements storage.Write. It uses a swap file and rename operation to ensure the change is
// atomic.
func (f *File) Write(bytes []byte) error {
	nf := f.Path + ".swp"
	nfh, err := os.OpenFile(nf, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	_, err = nfh.Write(bytes)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	if err := nfh.Close(); err != nil {
		return fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	if err := os.Rename(nf, f.Path); err != nil {
		return fmt.Errorf("%w: %s", ErrStorageFailure, err)
	}

	return nil
}
