package dev_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/api/dev"
	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/andrewhowdencom/x40.link/uid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEnricher(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		generator *uid.Generator

		from, to *url.URL

		expected *url.URL
		err      error
	}{
		{
			name:      "no from host",
			generator: uid.New(uid.TypeFails),

			from: &url.URL{
				Path: "/foo",
			},
			to: &url.URL{Host: "example.local", Path: "/"},

			expected: &url.URL{
				Host: "x40.local",
				Path: "/foo",
			},

			err: nil,
		},
		{
			name:      "no path",
			generator: uid.New(uid.TypeStatic),

			from: &url.URL{
				Host: "x40.local",
			},
			to: &url.URL{Host: "example.local", Path: "/"},

			expected: &url.URL{
				Host: "x40.local",
				Path: "/6SCxiHS",
			},
		},
		{
			name:      "failed to generate",
			generator: uid.New(uid.TypeFails),
			from: &url.URL{
				Host: "x40.local",
			},
			to: &url.URL{Host: "example.local", Path: "/"},

			expected: &url.URL{
				Host: "x40.local",
			},
			err: uid.ErrFailed,
		},
		{
			name:      "no enrichment required",
			generator: uid.New(uid.TypeFails),
			from: &url.URL{
				Host: "x40.local",
				Path: "/foo",
			},
			to: &url.URL{Host: "example.local", Path: "/"},

			expected: &url.URL{
				Host: "x40.local",
				Path: "/foo",
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e := &dev.URLEnricher{
				Domain: "x40.local",
				Path:   tc.generator,
			}

			err := e.Enrich(tc.from, tc.to)

			assert.Equal(t, tc.expected, tc.from)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestGetURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		// Used to construct the interface
		str storage.Storer
		req *gendev.GetRequest

		resp *gendev.Response
		code codes.Code
	}{
		{
			name: "bad url",

			str: test.New(),
			req: &gendev.GetRequest{
				Url: "\x00",
			},

			resp: nil,
			code: codes.InvalidArgument,
		},
		{
			name: "url not found",
			str:  test.New(),
			req: &gendev.GetRequest{
				Url: "https://example.local",
			},

			resp: nil,
			code: codes.NotFound,
		},
		{
			name: "unauthorized",
			str:  test.New(test.WithError(storage.ErrUnauthorized)),
			req: &gendev.GetRequest{
				Url: "https://example.local",
			},

			resp: nil,
			code: codes.PermissionDenied,
		},
		{
			name: "storage failure",
			str:  test.New(test.WithError(errors.New("b0rked"))),
			req: &gendev.GetRequest{
				Url: "https://example.local",
			},
			resp: nil,
			code: codes.Internal,
		},
		{
			name: "all ok",
			str: func() storage.Storer {
				test := test.New()
				if err := test.Put(
					context.Background(),
					&url.URL{Scheme: "https", Host: "example.local", Path: "/"},
					&url.URL{Scheme: "https", Host: "example.local", Path: "/"},
				); err != nil {
					panic("problem setting up test case: " + err.Error())
				}

				return test
			}(),
			req: &gendev.GetRequest{
				Url: "https://example.local/",
			},
			resp: &gendev.Response{
				Url: "https://example.local/",
			},
			code: codes.OK,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &dev.URL{
				Storer: tc.str,
			}

			resp, err := srv.Get(context.Background(), tc.req)

			assert.Equal(t, tc.resp, resp)

			// nil error is codes.OK and isStatus.
			status, isStatus := status.FromError(err)
			assert.True(t, isStatus)
			assert.Equal(t, tc.code, status.Code())
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		str storage.Storer
		en  func(from *url.URL, to *url.URL) error

		req  *gendev.NewRequest
		resp *gendev.Response

		code codes.Code
	}{
		{
			name: "missing req",
			str:  test.New(),
			en: (&dev.URLEnricher{
				Domain: "x40.local",
				Path:   uid.New(uid.TypeStatic),
			}).Enrich,
			req: &gendev.NewRequest{
				SendTo: "https://example.local",
			},
			resp: &gendev.Response{
				Url: "//x40.local/6SCxiHS",
			},
		},
		{
			name: "bad destination url",
			str:  test.New(),
			en:   func(from, to *url.URL) error { return nil },
			req: &gendev.NewRequest{
				SendTo: "\x00",
			},

			code: codes.InvalidArgument,
		},
		{
			name: "unauthorized",
			str:  test.New(test.WithError(storage.ErrUnauthorized)),
			en:   func(from, to *url.URL) error { return nil },
			req: &gendev.NewRequest{
				On: &gendev.RedirectOn{
					Host: "example.local",
					Path: "/",
				},
				SendTo: "https://example.local/2",
			},

			code: codes.PermissionDenied,
		},
		{
			name: "failure to write",
			str:  test.New(test.WithError(errors.New("b0rked"))),
			en:   func(from, to *url.URL) error { return nil },
			req: &gendev.NewRequest{
				On: &gendev.RedirectOn{
					Host: "example.local",
					Path: "/",
				},
				SendTo: "https://example.local/2",
			},
			code: codes.Internal,
		},
		{
			name: "no polyfilling required",
			str:  test.New(),
			en:   func(from, to *url.URL) error { return nil },
			req: &gendev.NewRequest{
				On: &gendev.RedirectOn{
					Host: "example.local",
					Path: "/",
				},
				SendTo: "https://example.local/2",
			},
			resp: &gendev.Response{
				Url: "//example.local/",
			},
			code: codes.OK,
		},
		{
			name: "enricher fails",
			str:  test.New(),
			en: func(from, to *url.URL) error {
				return fmt.Errorf("enricher fails")
			},
			req: &gendev.NewRequest{
				On: &gendev.RedirectOn{
					Host: "example.local",
					Path: "/",
				},
				SendTo: "https://example.local/2",
			},
			resp: nil,
			code: codes.Internal,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &dev.URL{
				Storer:   tc.str,
				Enricher: tc.en,
			}
			resp, err := srv.New(context.Background(), tc.req)

			assert.Equal(t, tc.resp, resp)

			// nil error is codes.OK and isStatus.
			status, isStatus := status.FromError(err)
			assert.True(t, isStatus)
			assert.Equal(t, tc.code, status.Code())
		})
	}
}
