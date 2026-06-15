package main

import (
	"context"
	"errors"
	"testing"

	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/sysexits"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeClient is a minimal api.Client implementation for testing doResolveWithClient. Only
// the Get method is configured per-test; New is left as a panic so we can detect any
// accidental use.
type fakeClient struct {
	get func(ctx context.Context, in *gendev.GetRequest, opts ...grpc.CallOption) (*gendev.Response, error)
}

func (f *fakeClient) Get(ctx context.Context, in *gendev.GetRequest, opts ...grpc.CallOption) (*gendev.Response, error) {
	return f.get(ctx, in, opts...)
}

func (f *fakeClient) New(_ context.Context, _ *gendev.NewRequest, _ ...grpc.CallOption) (*gendev.Response, error) {
	panic("fakeClient.New invoked; doResolveWithClient should not call New")
}

func TestDoResolveWithClient(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string

		// Setup the fake client to return this response/error.
		clientResp *gendev.Response
		clientErr  error

		// The input to doResolveWithClient.
		input string

		// What we expect.
		expectedURL   string
		expectedError error // Use errors.Is for matching; nil for "no error expected"
	}{
		{
			name: "valid input returns destination",

			clientResp: &gendev.Response{
				Url: "https://destination.example/path",
			},
			clientErr: nil,

			input: "https://x40.link/abc",

			expectedURL:   "https://destination.example/path",
			expectedError: nil,
		},
		{
			name: "input without scheme is treated as https",

			clientResp: &gendev.Response{
				Url: "https://destination.example/path",
			},
			clientErr: nil,

			input: "x40.link/abc",

			expectedURL:   "https://destination.example/path",
			expectedError: nil,
		},
		{
			name: "destination with // prefix has prefix stripped",

			clientResp: &gendev.Response{
				Url: "//destination.example/path",
			},
			clientErr: nil,

			input: "https://x40.link/abc",

			expectedURL:   "destination.example/path",
			expectedError: nil,
		},
		{
			name: "not found returns DataErr-wrapped error",

			clientResp: nil,
			clientErr:  status.Error(codes.NotFound, "url not found"),

			input: "https://x40.link/abc",

			expectedURL:   "",
			expectedError: sysexits.DataErr,
		},
		{
			name: "invalid argument returns DataErr-wrapped error",

			clientResp: nil,
			clientErr:  status.Error(codes.InvalidArgument, "url parse failure"),

			input: "https://x40.link/abc",

			expectedURL:   "",
			expectedError: sysexits.DataErr,
		},
		{
			name: "other gRPC error returns Protocol-wrapped error",

			clientResp: nil,
			clientErr:  status.Error(codes.Internal, "boom"),

			input: "https://x40.link/abc",

			expectedURL:   "",
			expectedError: sysexits.Protocol,
		},
		{
			name: "transport-level error returns NoHost-wrapped error",

			clientResp: nil,
			clientErr:  errors.New("connection refused"),

			input: "https://x40.link/abc",

			expectedURL:   "",
			expectedError: sysexits.NoHost,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fc := &fakeClient{
				get: func(_ context.Context, _ *gendev.GetRequest, _ ...grpc.CallOption) (*gendev.Response, error) {
					return tc.clientResp, tc.clientErr
				},
			}

			got, err := doResolveWithClient(context.Background(), fc, tc.input)

			assert.Equal(t, tc.expectedURL, got)
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectedError),
					"expected error chain to contain %v, got %v", tc.expectedError, err)
			}
		})
	}
}
