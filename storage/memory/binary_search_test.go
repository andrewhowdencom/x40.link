package memory

import (
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBinarySearchPutOrder validates that search can indeed insert content in an ordered set, else
// there is no point in querying it.
func TestBinarySearchPutOrder(t *testing.T) {
	t.Parallel()

	bs := NewBinarySearch()
	slugs := []string{"Iu1kie3N", "Thao6aef", "uC8as2oo", "Ba7tu9oh", "Za5maigh", "Sheepie4", "Jas6ceew", "Bohwi4Mi",
		"heeQuoh6", "amuWei5A", "ohC9Aip6", "cooC3Ies", "Hu0Saiz0", "Noomah7a", "viepha0E", "amoa9oJi", "ahQu8oos",
		"aiquu4Ev", "Aing2OhD", "YooJooz7"}

	for _, slug := range slugs {
		err := bs.Put(&url.URL{
			Host: "s3k", Path: "/" + slug,
		}, &url.URL{
			Host: "andrewhowden.com", Path: "/tests",
		})

		assert.Nil(t, err)
	}

	slices.Sort(slugs)

	for i := range slugs {
		assert.Equal(t, "/"+slugs[i], bs.idx[i].from.Path)
	}
}
