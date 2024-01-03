package yaml_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/andrewhowdencom/x40.link/storage/yaml"
	"github.com/stretchr/testify/assert"
)

func TestNewYaml(t *testing.T) {
	te := errors.New("I'm the test error")

	type urls struct {
		f, t *url.URL
		err  error
	}

	for _, tc := range []struct {
		name string

		str storage.Storer
		in  io.Reader

		err  error
		urls []urls
	}{
		{
			name: "normal, average and healthy",
			str:  memory.NewHashTable(),
			in: bytes.NewBufferString(`
---
- from: //x40/foo
  to: //k3s/bar
- from: //x40/bar
  to: //k3s/baz
`),
			urls: []urls{
				{
					f: &url.URL{Host: "x40", Path: "/foo"},
					t: &url.URL{Host: "k3s", Path: "/bar"},
				},
			},
		},
		{
			name: "single line corrupt (tab character)",
			str:  memory.NewHashTable(),
			in: bytes.NewBufferString(`
- from: "//	/foo"
  to: //k3s/bar
- from: //x40/bar
  to: //k3s/baz				
`),
			urls: []urls{
				{
					f:   &url.URL{Host: "	", Path: "/foo"},
					err: storage.ErrNotFound,
				},
			},
		},
		{
			name: "yaml corrupt",
			str:  memory.NewHashTable(),
			in:   bytes.NewBufferString(`I'm not yaml!`),
			err:  storage.ErrStorageSetupFailed,
		},
		{
			name: "storage failure",
			str:  test.New(test.WithError(te)),
			in: bytes.NewBufferString(`
---
- from: //x40/foo
  to: //k3s/bar
- from: //x40/bar
  to: //k3s/baz
`),
			err: storage.ErrStorageSetupFailed,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			y, err := yaml.New(tc.str, tc.in)

			// Validate the error case
			assert.ErrorIs(t, err, tc.err)

			// Validate State
			for _, u := range tc.urls {
				nu, err := y.Get(u.f)
				assert.ErrorIs(t, err, u.err)
				assert.Equal(t, u.t, nu)
			}
		})
	}
}

func TestYamlRejectWrite(t *testing.T) {
	t.Parallel()
	y, err := yaml.New(memory.NewHashTable(), bytes.NewBufferString("---"))

	assert.Nil(t, err)

	err = y.Put(
		&url.URL{Host: "x40", Path: "/foo"},
		&url.URL{Host: "k3s", Path: "/bar"},
	)

	assert.ErrorIs(t, err, storage.ErrReadOnlyStorage)
}

func TestLoggerOverride(t *testing.T) {
	exist := yaml.Log
	defer func() { yaml.Log = exist }()

	// Set up a new logger
	b := &bytes.Buffer{}
	yaml.Log = slog.New(slog.NewTextHandler(b, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Get an error condition (failed URL). Return values are discarded as they are (implicitly) validated via the
	// log assertion.
	_, _ = yaml.New(memory.NewHashTable(), bytes.NewBufferString(`
- from: "//	/foo"
  to: //k3s/bar
`))

	// Validate the output
	assert.Contains(t, b.String(), "invalid control character")
}
