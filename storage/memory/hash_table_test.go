package memory_test

import (
	"fmt"
	"log"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/stretchr/testify/assert"
)

// ExampleNewHashTable describes how to use the hash table based lookup to query what the supplied URL
// would be.
func ExampleNewHashTable() {
	// Consider the following several URLs
	// x40/a → andrewhowden.com/a
	// x40/n → andrewhowden.com/longer-n
	// x40.link/f00 → google.com

	ht := memory.NewHashTable()

	// Insert the URLs into the hash table based lookup
	for _, tu := range []struct {
		f, t string
	}{
		{f: "x40/a", t: "andrewhowden.com/a"},
		{f: "x40/n", t: "andrewhowden.com/longer-n"},
		{f: "x40.link/f00", t: "google.com"},
	} {
		// Normally, we should be handling errors from both the URL parsing as well as attempting to write
		// to storage. The hashmap has no failure modes, but we should not rely on this being persistent
		// indefinitely into the future (e.g. collisions)
		from, _ := url.Parse(tu.f)
		to, _ := url.Parse(tu.t)

		err := ht.Put(from, to)
		if err != nil {
			log.Println(err)
		}
	}

	// Lookup a value supplied by the user
	l, _ := url.Parse("x40/a")

	// Normally, we should handle the error from the fetch operation (e.g. not found)
	ret, _ := ht.Get(l)
	fmt.Println(ret.String())
	// Output: andrewhowden.com/a
}

// TestNewHashTable simply validates that the hash table returns something that will not panic when used.
func TestNewHashTable(t *testing.T) {
	t.Parallel()

	ht := memory.NewHashTable()
	assert.Nil(t, ht.Put(&url.URL{
		Host: "x40",
	}, &url.URL{
		Host: "andrewhowden.com",
	}))

	res, err := ht.Get(&url.URL{
		Host: "x40",
	})

	assert.Nil(t, err)
	assert.Equal(t, &url.URL{
		Host: "andrewhowden.com",
	}, res)
}
