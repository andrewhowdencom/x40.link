package memory_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/s3k.link/storage/memory"
	"github.com/stretchr/testify/assert"
)

// ExampleNewHashTable describes how to use the hash table based lookup to query what the supplied URL
// would be.
func ExampleNewHashTable() {
	// Consider the following several URLs
	// s3k/a → andrewhowden.com/a
	// s3k/n → andrewhowden.com/longer-n
	// s3k.link/f00 → google.com

	ht := memory.NewHashTable()

	// Insert the URLs into the hash table based lookup
	for _, tu := range []struct {
		f, t string
	}{
		{f: "s3k/a", t: "andrewhowden.com/a"},
		{f: "s3k/n", t: "andrewhowden.com/longer-n"},
		{f: "s3k.link/f00", t: "google.com"},
	} {
		// Normally, we should be handling errors from both the URL parsing as well as attempting to write
		// to storage. The hashmap has no failure modes, but we should not rely on this being persistent
		// indefinitely into the future (e.g. collisions)
		from, _ := url.Parse(tu.f)
		to, _ := url.Parse(tu.t)

		ht.Put(from, to)
	}

	// Lookup a value supplied by the user
	l, _ := url.Parse("s3k/a")

	// Normally, we should handle the error from the fetch operation (e.g. not found)
	ret, _ := ht.Get(l)
	fmt.Println(ret.String())
	// Output: andrewhowden.com/a
}

// TestNewHashTable simply validates that the hash table returns something that will not panic when used.
func TestNewHashTable(t *testing.T) {
	t.Parallel()

	ht := memory.NewHashTable()
	ht.Put(&url.URL{
		Host: "s3k",
	}, &url.URL{
		Host: "andrewhowden.com",
	})

	res, err := ht.Get(&url.URL{
		Host: "s3k",
	})

	assert.Nil(t, err)
	assert.Equal(t, &url.URL{
		Host: "andrewhowden.com",
	}, res)
}
