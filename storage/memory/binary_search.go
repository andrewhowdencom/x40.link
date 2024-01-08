package memory

import (
	"net/url"
	"sync"

	"github.com/andrewhowdencom/x40.link/storage"
)

// BinarySearch is an implementation of the search of a (sorted) data set that has much better performance
// than the linear search, on average. Best case, its O(1) if the record is (somehow) in the middle,
// worst case its O(log(n))
//
// TODO: This probably needs to be a linked list, rather than a slice. Slice will be enormously inefficient
// when we're talking about reallocating that dat.
type BinarySearch struct {
	idx []tu
	mu  sync.RWMutex
}

// NewBinarySearch initializes a binary search object with its own properties initialized
func NewBinarySearch() *BinarySearch {
	return &BinarySearch{
		idx: make([]tu, 0),
	}
}

// find is a shared function that allow determining either where we need to insert the value, or if the value exists,
// where to update it. It works by defining boundary conditions in which to search (an upper and a lower), and
// as binary search progresses, reducing the bounds by half — thereby cutting the result set in half.
func (bs *BinarySearch) find(in *url.URL) (found bool, nearest int) {
	// If there's nothing setup just yet, there can be nothing in the set. Return.
	if len(bs.idx) == 0 {
		return false, 0
	}

	// Define upper and lower bounds to search within
	lower := 0
	upper := len(bs.idx)
	next := (upper - lower) / 2

	// The loop
	for {
		// We've found the result
		if in.String() == bs.idx[next].from.String() {
			return true, next
		}

		// We've not found the result, but we're also not going to. The search has run its course.
		if (upper - lower) == 1 {
			return false, lower
		}

		// If the requested sample is (lexigraphically) higher than the input, search in the "right hand" half
		// of the stack.
		if in.String() > bs.idx[next].from.String() {
			lower = lower + ((upper - lower) / 2)
		} else {
			upper = upper - ((upper - lower) / 2)
		}

		// Set the pointer to the next value to be mid way between the lower and mid boundaries.
		next = lower + ((upper - lower) / 2)
	}
}

// Get returns an URL, given an input URL
func (bs *BinarySearch) Get(in *url.URL) (*url.URL, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	found, pos := bs.find(in)
	if found {
		return bs.idx[pos].to, nil
	}

	return nil, storage.ErrNotFound
}

// Put writes the record into the set. Takes responsibility for determining the position in which to add the
// new value, so that the underlying set retains order.
func (bs *BinarySearch) Put(f *url.URL, t *url.URL) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Special case: If the array is empty, just start it.
	if len(bs.idx) == 0 {
		bs.idx = append(bs.idx, tu{from: f, to: t})
		return nil
	}

	found, pos := bs.find(f)

	// If the record is already there, update it.
	if found {
		bs.idx[pos].to = t
		return nil
	}

	// If the input is larger than the existing item at that address, we want to put the new input to the
	// right of that address.
	//
	// Otherwise, we want to put the input in the existing address, shifting everything thats there
	// to the right.
	if f.String() > bs.idx[pos].from.String() {
		pos = pos + 1
	}

	// There's no nice way to composit the array while retaining order. Given this, a new slice is created with the
	// same values, and it replaces the old slice. This is definitely not the most memory efficient approach, as the
	// new slice will need to be reallocated.
	ni := []tu{}
	ni = append(ni, bs.idx[0:pos]...)
	ni = append(ni, tu{from: f, to: t})
	ni = append(ni, bs.idx[pos:]...)

	bs.idx = ni

	return nil
}
