package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/server"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewServer_WithBadOption(t *testing.T) {
	t.Parallel()

	_, err := server.New(func(s *http.Server) error {
		return errors.New("i am bad")
	})

	assert.ErrorIs(t, err, server.ErrFailedToApplyOption)
}

func TestNewServer_WithListenAddress(t *testing.T) {
	t.Parallel()

	srv, err := server.New(server.WithListenAddress("0.0.0.0:1234"))

	assert.Equal(t, "0.0.0.0:1234", srv.Addr)
	assert.Nil(t, err)
}

func TestNewServer_WithDefaults(t *testing.T) {
	t.Parallel()

	srv, err := server.New()
	assert.Nil(t, err)

	mux, ok := srv.Handler.(*chi.Mux)
	assert.True(t, ok)

	// This is a weak test, but not sure yet how to validate this
	assert.Len(t, mux.Middlewares(), 2)
}

func TestNewServer_WithMiddleware(t *testing.T) {
	t.Parallel()

	iWasInvoked := false

	srv, err := server.New(server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			iWasInvoked = true

			next.ServeHTTP(w, r)
		})
	}))

	assert.Nil(t, err)

	// Create a path so the request actually gets routed somewhere
	c := srv.Handler.(*chi.Mux)
	c.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Yeah!"))
	})

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)

	srv.Handler.ServeHTTP(w, r)

	assert.True(t, iWasInvoked)
	assert.Equal(t, []byte("Yeah!"), w.Body.Bytes())
}

// WithStorage is a more extensive test as it binds a slug, rather than just modifying the state of http.Request
func TestNewServer_WithStorage(t *testing.T) {
	t.Parallel()

	storage := test.New()
	err := storage.Put(&url.URL{
		Host: "test",
		Path: "/foo",
	},
		&url.URL{
			Host: "test",
			Path: "/bar",
		},
	)
	assert.Nil(t, err)

	srv, err := server.New(server.WithStorage(storage))
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/foo", nil)
	req.Host = "test"

	srv.Handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Result().StatusCode)
	assert.Equal(t, "//test/bar", w.Header().Get("Location"))
}
