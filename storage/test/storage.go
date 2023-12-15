package test

import (
	"encoding/base64"
	"math/rand"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/s3k.link/storage"
)

// A seed that is sufficiently random that the tests make sense, but sufficiently stable they can be repeated
// over multiple runs.
const seed = 42

func BenchmarkStorage(b *testing.B, str storage.Storer, iter int64) {
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

// RaceStorage is designed to stress the storage by using it concurrently, such that the go race detector can
// figure out if variables are being shared across the stack.
func RaceStorage(str storage.Storer) {

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
