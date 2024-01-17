package uid_test

import (
	"math/big"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/uid"
	"github.com/stretchr/testify/assert"
)

// TestRand just tests whether rand generates any value.
func TestRand(t *testing.T) {
	t.Parallel()

	// Generate the number
	g := uid.New(uid.TypeRandom)
	v, err := g.ID(&url.URL{})

	assert.Nil(t, err)
	assert.NotEqual(t, "", v)

	// Convert back to its number, and validate the first byte is the prefix.
	var i big.Int
	i.SetString(v, 62)
	assert.Equal(t, uid.TypeRandom, i.Bytes()[0])
}
