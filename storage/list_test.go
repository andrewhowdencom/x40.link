package storage_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/boltdb"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/stretchr/testify/require"
)

func TestListUnownedStorage(t *testing.T) {
	for name, newStore := range map[string]func(*testing.T) storage.Storer{
		"hash":   func(*testing.T) storage.Storer { return memory.NewHashTable() },
		"linear": func(*testing.T) storage.Storer { return memory.NewLinearSearch() },
		"binary": func(*testing.T) storage.Storer { return memory.NewBinarySearch() },
		"bolt": func(t *testing.T) storage.Storer {
			store, err := boltdb.New(t.TempDir() + "/links.db")
			require.NoError(t, err)
			return store
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			for _, source := range []string{"//b.example/two", "//a.example/one", "//a.example/three"} {
				from, err := url.Parse(source)
				require.NoError(t, err)
				require.NoError(t, store.Put(ctx, from, &url.URL{Host: "target.example"}))
			}

			all, err := store.List(ctx, "")
			require.NoError(t, err)
			require.Equal(t, []string{"//a.example/one", "//a.example/three", "//b.example/two"}, sources(all))

			filtered, err := store.List(ctx, "a.example")
			require.NoError(t, err)
			require.Equal(t, []string{"//a.example/one", "//a.example/three"}, sources(filtered))
		})
	}
}

func sources(links []storage.Link) []string {
	result := make([]string, 0, len(links))
	for _, link := range links {
		result = append(result, link.From.String())
	}
	return result
}
