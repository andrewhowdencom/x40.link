package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/stretchr/testify/assert"
)

func TestStoreHandler_Get(t *testing.T) {
	sentinel := errors.New("Sentinel")

	for _, tc := range []struct {
		// Meta
		name string

		// Input
		req     *http.Request
		storage storage.Storer

		// Response
		statusCode int
		headers    http.Header
	}{
		{
			name: "everything ok, record found",

			req: &http.Request{
				Host: "s3k",
				URL: &url.URL{
					Path: "/foo",
				},
			},
			storage: func() storage.Storer {
				str := test.New()
				test.Must(str.Put(
					context.Background(),
					&url.URL{Host: "s3k", Path: "/foo"},
					&url.URL{Scheme: "https", Host: "andrewhowden.com", Path: "/"},
				))

				return str
			}(),

			statusCode: http.StatusTemporaryRedirect,
			headers: http.Header{
				"Location": []string{"https://andrewhowden.com/"},
			},
		},
		{
			name: "record missing",

			req: &http.Request{
				Host: "s3k",
				URL: &url.URL{
					Path: "/foo",
				},
			},
			storage: test.New(),
			headers: http.Header{},

			statusCode: http.StatusNotFound,
		},
		{
			name: "storage failure",

			req: &http.Request{
				Host: "s3k",
				URL: &url.URL{
					Path: "/foo",
				},
			},
			storage: test.New(test.WithError(sentinel)),
			headers: http.Header{},

			statusCode: http.StatusInternalServerError,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Bootstrap
			w := httptest.NewRecorder()

			handler := &strHandler{str: tc.storage}

			handler.Redirect(w, tc.req)

			assert.Equal(t, tc.statusCode, w.Result().StatusCode)
			// Headers is a partial check — we only assert on the keys
			// listed in tc.headers, not on the full set, since the
			// error path writes Content-Type via the problem document.
			for k, v := range tc.headers {
				assert.Equal(t, v, w.Result().Header[k])
			}
		})
	}

	// Silence unused-import warning for sentinel, which would otherwise
	// trigger if the test cases no longer reference it.
	_ = sentinel
}