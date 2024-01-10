package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/stretchr/testify/assert"
	"schneider.vip/problem"
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
		err        error
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
					&url.URL{Host: "s3k", Path: "/foo"},
					&url.URL{Scheme: "https", Host: "andrewhowden.com", Path: "/"},
				))

				return str
			}(),

			statusCode: http.StatusTemporaryRedirect,
			headers: http.Header{
				"Location": []string{"https://andrewhowden.com/"},
			},
			err: nil,
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
			err: problem.New(
				problem.Status(http.StatusNotFound),
				problem.Custom("url", "//s3k/foo"),
			),
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

			statusCode: http.StatusInternalServerError,
			headers:    http.Header{},
			err:        sentinel,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Bootstrap
			w := httptest.NewRecorder()

			handler := &strHandler{str: tc.storage}

			handler.Redirect(w, tc.req)

			err, isError := tc.req.Context().Value(CtxErrors).(error)

			if tc.err == nil {
				assert.False(t, isError)
				assert.Equal(t, tc.statusCode, w.Result().StatusCode)
				assert.Equal(t, w.Result().Header, tc.headers)
			} else {
				assert.True(t, isError)
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}
}
