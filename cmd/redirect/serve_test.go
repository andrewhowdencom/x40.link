package redirect

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetStorage tests flag wiring to configuration, as well as configuration wiring to functions that
// use it.
func TestGetStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "TestGetStorage")
	assert.Nil(t, err)

	for _, tc := range []struct {
		name string

		fName, fVal string
		err         error

		setup, teardown func()
	}{
		{
			name:  "in memory storage",
			fName: flagStrHashMap,
			fVal:  "true",
		},
		{
			name:  "disabled in memory storage",
			fName: flagStrHashMap,
			fVal:  "false",
			err:   ErrFailedStorageSetup,
		},
		{
			name:  "yaml storage",
			fName: flagStrYAML,
			fVal:  tmpDir + "/urls.yaml",
			setup: func() {
				file, err := os.Create(tmpDir + "/urls.yaml")
				if err != nil {
					panic(err)
				}

				if _, err := file.Write([]byte(`
---
- from: //x40/foo
  to: //k3s/bar
- from: //x40/bar
  to: //k3s/baz
`)); err != nil {
					panic(err)
				}

				if err := file.Close(); err != nil {
					panic(err)
				}
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Allow setting up whatever resources need to exist for the storage (e.g. temporary directories to
			// store content)
			if tc.setup != nil {
				tc.setup()
			}
			if tc.teardown != nil {
				defer tc.teardown()
			}

			// Replicate the supplied user option.
			assert.Nil(t, serveFlagSet.Set(tc.fName, tc.fVal))

			_, err := getStorage(serveFlagSet)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
