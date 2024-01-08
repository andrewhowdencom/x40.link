package storage_test

import (
	"encoding/base64"
	"math/rand"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/boltdb"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/stretchr/testify/assert"
)

// A seed that is sufficiently random that the tests make sense, but sufficiently stable they can be repeated
// over multiple runs.
const seed = 42

// Factories to generate valid storage engines
var sinkFactories = map[string]func(string) storage.Storer{
	"hash table":    func(string) storage.Storer { return memory.NewHashTable() },
	"linear search": func(string) storage.Storer { return memory.NewLinearSearch() },
	"binary search": func(string) storage.Storer { return memory.NewBinarySearch() },
	"boltdb": func(n string) storage.Storer {
		db, err := boltdb.New(path.Join(os.TempDir(), "test+"+n+"+url-shortner.db"))
		if err != nil {
			panic(err)
		}

		return db
	},
}

// Factories to tear down valid storage engines
var teardownFunc = map[string]func(string){
	"hash table":    func(string) {},
	"linear search": func(string) {},
	"binary search": func(string) {},
	"boltdb": func(n string) {
		if err := os.Remove(path.Join(os.TempDir(), "test+"+n+"+url-shortner.db")); err != nil {
			panic(err)
		}
	},
}

// benchmark is a generic approach to benchmarking the various different storage implementations at different underlying data
// sizes.
//
// All enginers are benchmarked against this approach.
//
// TODO: Add additional benchmarks for:
// 1. Write
// 2. Not Found
func benchmark(b *testing.B, str storage.Storer, iter int) {
	// Generate the URLs randomly. Uses Rand.Read() and Base64 URL safe encoding to generate
	// "fairly random" URLs, creating a large, unsorted array. All URLs point to the same result, as this is
	// outside the scope of the benchmark.
	urls := []*url.URL{}
	dest := &url.URL{
		Host: "andrewhowden.com",
		Path: "/benchmarks",
	}

	rand := rand.New(rand.NewSource(seed))
	for i := 0; i <= iter; i++ {

		bytes := make([]byte, 10)
		rand.Read(bytes)

		next := &url.URL{
			Host: "x40",
			Path: base64.URLEncoding.EncodeToString(bytes),
		}
		urls = append(urls, next)

		if err := str.Put(next, dest); err != nil {
			b.Log(err)
			b.FailNow()
		}
	}

	// Iterate through the whole list, finding them all. The actual benchmark.
	b.ResetTimer()
	for _, u := range urls {
		if _, err := str.Get(u); err != nil {
			b.Log(err)
			b.FailNow()
		}
	}
}

// race is designed to stress the storage by using it concurrently, such that the go race detector can
// figure out if variables are being shared across the stack.
func race(str storage.Storer) {
	for i := 0; i < 1000; i++ {
		go func() {
			// If the number is divisible by 4 (which it should be, 25% of the time) then make it a write operation.
			if rand.Int()%4 == 0 {
				if err := str.Put(&url.URL{
					Host: "x40",
				}, &url.URL{
					Host: "k3s",
				}); err != nil {
					panic(err)
				}
			} else {
				if _, err := str.Get(&url.URL{
					Host: "x40",
				}); err != nil && err != storage.ErrNotFound {
					panic(err)
				}
			}
		}()
	}
}

// BenchmarkAll benchmarkes all storage implementations (supplied by the sinkFactories variable)
func BenchmarkAll(b *testing.B) {
	// Some of the implementations are more efficient than others. It is time prohibitive to run the benchmarks
	// on the less efficient ones, so they're skipped.
	benchmarkLengths := map[string][]int{
		"hash table":    {10, 100, 1000, 100000},
		"linear search": {10, 100, 1000},
		"binary search": {10, 100, 1000},
		"boltdb":        {10, 100, 1000},
	}

	for n, f := range sinkFactories {
		f := f
		n := n

		for _, l := range benchmarkLengths[n] {
			l := l

			b.Run(n+"+"+strconv.Itoa(int(l)), func(b *testing.B) {
				name := "benchmark+" + strconv.Itoa(int(l))
				defer teardownFunc[n](name)

				benchmark(b, f(name), l)
			})
		}
	}
}

// TestRaceAll tests the concurrency of all implementations (with the go test -race flag on)
func TestRaceAll(t *testing.T) {
	for n, f := range sinkFactories {
		n := n
		f := f
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			defer teardownFunc[n]("race")
			race(f("race"))
		})
	}
}

// TestComplianceAll tests that the storages actually store and retrieve valid records in the (simplest) expected ways.
func TestComplianceAll(t *testing.T) {
	for n, f := range sinkFactories {
		f := f
		n := n

		t.Run(n, func(t *testing.T) {
			t.Parallel()

			str := f("compliance")
			defer teardownFunc[n]("compliance")

			// Query for a record that doesn't exit, to ensure the data store will not panic.
			_, err := str.Get(&url.URL{Host: "x40"})
			assert.ErrorIs(t, err, storage.ErrNotFound)

			// Insert and query a record.
			assert.Nil(t, str.Put(&url.URL{Host: "x40"}, &url.URL{Host: "andrewhowden.com"}))

			res, err := str.Get(&url.URL{
				Host: "x40",
			})

			assert.Nil(t, err)
			assert.Equal(t, &url.URL{
				Host: "andrewhowden.com",
			}, res)
		})
	}
}
