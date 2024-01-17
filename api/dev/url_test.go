package dev_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/andrewhowdencom/x40.link/api/dev"
	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/test"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		// Used to construct the interface
		str storage.Storer
		req *gendev.Request

		resp *gendev.Response
		code codes.Code
	}{
		{
			name: "bad url",

			str: test.New(),
			req: &gendev.Request{
				Url: "\x00",
			},

			resp: nil,
			code: codes.InvalidArgument,
		},
		{
			name: "url not found",
			str:  test.New(),
			req: &gendev.Request{
				Url: "https://example.local",
			},

			resp: nil,
			code: codes.NotFound,
		},
		{
			name: "unauthorized",
			str:  test.New(test.WithError(storage.ErrUnauthorized)),
			req: &gendev.Request{
				Url: "https://example.local",
			},

			resp: nil,
			code: codes.PermissionDenied,
		},
		{
			name: "storage failure",
			str:  test.New(test.WithError(errors.New("b0rked"))),
			req: &gendev.Request{
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
			req: &gendev.Request{
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

func TestNewCustom(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		str storage.Storer
		req *gendev.CustomRequest

		code codes.Code
	}{
		{
			name: "bad from url",
			str:  nil,
			req: &gendev.CustomRequest{
				From: "\x00",
				To:   "https://example.local",
			},

			code: codes.InvalidArgument,
		},
		{
			name: "bad to url",

			str: nil,
			req: &gendev.CustomRequest{
				From: "https://example.local",
				To:   "\x00",
			},

			code: codes.InvalidArgument,
		},
		{
			name: "unauthorized",
			str:  test.New(test.WithError(storage.ErrUnauthorized)),
			req: &gendev.CustomRequest{
				From: "https://example.local",
				To:   "https://example.local/2",
			},

			code: codes.PermissionDenied,
		},
		{
			name: "failure to write",
			str:  test.New(test.WithError(errors.New("b0rked"))),
			req: &gendev.CustomRequest{
				From: "https://example.local",
				To:   "https://example.local/2",
			},
			code: codes.Internal,
		},
		{
			name: "all ok",
			str:  test.New(),
			req: &gendev.CustomRequest{
				From: "https://example.local",
				To:   "https://example.local/2",
			},
			code: codes.OK,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

		})
	}
}
