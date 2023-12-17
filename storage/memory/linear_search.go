package memory

import (
	"net/url"
	"sync"

	"github.com/andrewhowdencom/s3k.link/storage"
)

// LinearSearch is an implementation of "worst case" search through the data. It has O(n) performance (i.e. for each
// new bit of data, the worst-case compelxity & performance of the lookup fails). Additionally, this
// is especially bad as "not found" requires iterating through the whole set.
type LinearSearch struct {
	idx []tu

	mu sync.RWMutex
}

// tu or "tuple"
type tu struct {
	from *url.URL
	to   *url.URL
}

func NewLinearSearch() *LinearSearch {
	return &LinearSearch{
		idx: make([]tu, 0),
	}
}

func (s *LinearSearch) Get(in *url.URL) (*url.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, tu := range s.idx {
		if in.String() == tu.from.String() {
			return tu.to, nil
		}
	}

	return nil, storage.ErrNotFound
}

func (s *LinearSearch) Put(f *url.URL, t *url.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.idx = append(s.idx, tu{
		from: f, to: t,
	})

	return nil
}
