package memory

import (
	"context"
	"net/url"
	"sync"

	"github.com/andrewhowdencom/x40.link/storage"
)

// LinearSearch is an implementation of "worst case" search through the data. It has O(n) performance (i.e. for each
// new bit of data, the worst-case compelxity & performance of the lookup fails). Additionally, this
// is especially bad as "not found" requires iterating through the whole set.
type LinearSearch struct {
	idx []tu

	mu sync.RWMutex
}

// NewLinearSearch implements the most naive approach to querying data
func NewLinearSearch() *LinearSearch {
	return &LinearSearch{
		idx: make([]tu, 0),
	}
}

// Get queries the linear search. It just iterates through the whole slice.
func (s *LinearSearch) Get(_ context.Context, in *url.URL) (*url.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, tu := range s.idx {
		if in.String() == tu.from.String() {
			return tu.to, nil
		}
	}

	return nil, storage.ErrNotFound
}

// Put writes the URL into storage, appending it to the slice.
func (s *LinearSearch) Put(_ context.Context, f *url.URL, t *url.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.idx = append(s.idx, tu{
		from: f, to: t,
	})

	return nil
}

func (s *LinearSearch) List(_ context.Context, domain string) ([]storage.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	links := make([]storage.Link, 0)
	for _, item := range s.idx {
		if domain == "" || item.from.Host == domain {
			links = append(links, storage.Link{From: item.from, To: item.to})
		}
	}
	storage.SortLinks(links)
	return links, nil
}
