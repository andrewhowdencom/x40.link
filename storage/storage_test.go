package storage_test

import (
	"encoding/base64"
	"math/rand"
	"net/url"
	"strconv"
	"testing"

	"github.com/andrewhowdencom/s3k.link/storage"
	"github.com/andrewhowdencom/s3k.link/storage/memory"
)

// A seed that is sufficiently random that the tests make sense, but sufficiently stable they can be repeated
// over multiple runs.
const seed = 42

// The lengths to benchmark the application on
var benchmarkLengths = []int64{10, 100, 1000, 100000, 5000000}
var sinkFactories = map[string]func() storage.Storer{
	"hash table": func() storage.Storer { return memory.NewHashTable() },
}

// benchmark is a generic approach to benchmarking the various different storage implementations at different underlying data
// sizes.
//
// All enginers are benchmarked against this approach.
func benchmark(b *testing.B, str storage.Storer, iter int64) {
	// Generate the URLs randomly. Uses Rand.Read() and Base64 URL safe encoding to generate
	// "fairly random" URLs, creating a large, unsorted array. All URLs point to the same result, as this is
	// outside the scope of the benchmark.
	urls := []*url.URL{}
	dest := &url.URL{
		Host: "andrewhowden.com",
		Path: "/benchmarks",
	}

	rand := rand.New(rand.NewSource(seed))
	var i int64
	for i = 0; i <= iter; i++ {

		bytes := make([]byte, 10)
		rand.Read(bytes)

		next := &url.URL{
			Host: "s3k",
			Path: base64.URLEncoding.EncodeToString(bytes),
		}
		urls = append(urls, next)
		str.Put(next, dest)
	}

	// Iterate through the whole list, finding them all. The actual benchmark.
	b.ResetTimer()
	for _, u := range urls {
		str.Get(u)
	}
}

// race is designed to stress the storage by using it concurrently, such that the go race detector can
// figure out if variables are being shared across the stack.
func race(str storage.Storer) {

	for i := 0; i < 1000; i++ {
		go func() {
			// If the number is divisible by 4 (which it should be, 25% of the time) then make it a write operation.
			if rand.Int()%4 == 0 {
				str.Put(&url.URL{
					Host: "s3k",
				}, &url.URL{
					Host: "k3s",
				})
			} else {
				str.Get(&url.URL{
					Host: "s3k",
				})
			}
		}()
	}
}

// BenchmarkAll benchmarkes all storage implementations (supplied by the sinkFactories variable)
func BenchmarkAll(b *testing.B) {
	for n, f := range sinkFactories {
		for _, l := range benchmarkLengths {
			b.Run(n+"+"+strconv.Itoa(int(l)), func(b *testing.B) {
				benchmark(b, f(), l)
			})
		}
	}
}

// TestRaceAll tests the concurrency of all implementations (with the go test -race flag on)
func TestRaceAll(t *testing.T) {
	for n, f := range sinkFactories {
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			race(f())
		})
	}
}