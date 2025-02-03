package storage

import "sync"

// Test storage is just storage that is useful to make assertions in tests. Do not use it in production; there are
// better tools for everything you could possibly do with this.
type Test struct {
	bytes []byte

	mu *sync.RWMutex

	err interface{}
}

// WithReadError sets the error for the read function to return.
func WithReadError(in func(*Test) error) func(t *Test) {
	return func(t *Test) {
		t.err = in
	}
}

// WithWriteError sets the error for the write function to return
func WithWriteError(in func(*Test, []byte) error) func(t *Test) {
	return func(t *Test) {
		t.err = in
	}
}

// WithBytes sets the initial state of the storage.
func WithBytes(b []byte) func(t *Test) {
	return func(t *Test) {
		t.bytes = b
	}
}

// NewTest generates a new test storage
func NewTest(f ...func(t *Test)) *Test {
	nt := &Test{
		mu: &sync.RWMutex{},
	}

	for _, dof := range f {
		dof(nt)
	}

	return nt
}

// Read implements storage.Read
func (t *Test) Read() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if f, ok := t.err.(func(*Test) error); ok {
		if err := f(t); err != nil {
			return nil, err
		}
	}

	if len(t.bytes) == 0 {
		return nil, ErrNotFound
	}

	return t.bytes, nil
}

// Write implements storage.Write
func (t *Test) Write(input []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if f, ok := t.err.(func(*Test, []byte) error); ok {
		if err := f(t, input); err != nil {
			return err
		}
	}

	t.bytes = input
	return nil
}
