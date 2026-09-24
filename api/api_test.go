package api_test

import (
	"testing"

	"github.com/andrewhowdencom/x40.link/api"
	"github.com/stretchr/testify/assert"
)

func TestListHasPermission(t *testing.T) {
	assert.Equal(t,
		"api.x40.link/scopes/x40.dev.url.ManageURLs.List",
		api.X40Permissions()["/x40.dev.url.ManageURLs/List"],
	)
}
