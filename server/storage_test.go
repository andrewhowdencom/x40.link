package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/gin-gonic/gin"
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

			statusCode: http.StatusNotFound,
			headers:    http.Header{},
			err:        nil,
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
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = tc.req

			handler := &strHandler{str: tc.storage}
			handler.Redirect(ctx)

			// Gin, within each handler, doesn't necessarily flush. Flushing, in this case, includes setting the
			// status
			ctx.Writer.Flush()

			// Validate the error conditrion
			if tc.err == nil {
				assert.Len(t, ctx.Errors, 0)
			} else {
				assert.Len(t, ctx.Errors, 1)
				assert.ErrorIs(t, ctx.Errors[0], tc.err)
			}

			// Validate the response
			assert.Equal(t, tc.statusCode, w.Result().StatusCode)
			assert.Equal(t, w.Result().Header, tc.headers)
		})
	}
}
