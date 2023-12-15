package memory

import (
	"net/url"
	"sync"

	"github.com/andrewhowdencom/s3k.link/storage"
)

// HashTable stores the entire dataset within Go's implementation of a hash table (a map). It
// has O(1) complexity, as it is always looking up something well known within a finite space.
type HashTable struct {
	table map[string]*url.URL
	mu    sync.RWMutex
}

// NewHashTable initializes a new hash table, with the appropriate default values. It also exposes the hash
// table outside this package, without needing to expose its internal properties (e.g. the table and mutexes)
// and so on.
func NewHashTable() *HashTable {
	return &HashTable{
		table: make(map[string]*url.URL),
		mu:    sync.RWMutex{},
	}
}

// Get fetches a URL. It looks it up in the hashmap by converting it to a string representation (which should be
// unique), after which it will lookup the corresponding URL.
func (ht *HashTable) Get(in *url.URL) (*url.URL, error) {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	if v, ok := ht.table[in.String()]; ok {
		return v, nil
	}

	return nil, storage.ErrNotFound
}

// Storage writes a URL into memory. Designed to be used primarily via "loader" infrastructure, such as the
// YAML loader.
func (ht *HashTable) Put(f *url.URL, t *url.URL) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.table[f.String()] = t

	return nil
}
