package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithListenAddress(t *testing.T) {
	t.Parallel()

	srv, err := New(WithListenAddress("localhost:1234"))
	assert.Nil(t, err)
	assert.Equal(t, srv.listen, "localhost:1234")
}
