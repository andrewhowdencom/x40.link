package server_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/andrewhowdencom/x40.link/server"
	"github.com/stretchr/testify/assert"
)

func TestIsGRPC(t *testing.T) {
	t.Parallel()

	nr := func(hk, hv string, ver int) *http.Request {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Add(hk, hv)
		req.ProtoMajor = ver

		return req
	}

	for _, tc := range []struct {
		name string

		req *http.Request

		expected bool
	}{
		{
			name: "no match",
			req:  nr("accept", "application/json", 0),
		},
		{
			name: "yes application, no version",
			req:  nr("Content-Type", "application/grpc", 0),
		},
		{
			name: "no application, yes version",
			req:  nr("Content-Type", "application/json", 2),
		},
		{
			name:     "yes application, yes version",
			req:      nr("Content-Type", "application/grpc", 2),
			expected: true,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, server.IsGRPC(tc.req))
		})
	}
}

func TestH2C_Match(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		req  *http.Request

		expected bool
	}{
		{
			name: "should match",
			req: func() *http.Request {
				req, _ := http.NewRequest("PRI", "*", bytes.NewBufferString("SM\r\n\r\n"))
				req.Proto = "HTTP/2.0"

				return req
			}(),
			expected: true,
		},
		{
			name: "has wrong method",
			req: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)

				return req
			}(),
			expected: false,
		},
		{
			name: "has headers",
			req: func() *http.Request {
				req, _ := http.NewRequest("PRI", "*", bytes.NewBufferString("SM\r\n\r\n"))
				req.Header.Add("Header", "b0rk")

				return req
			}(),
			expected: false,
		},
		{
			name: "has wrong path",
			req: func() *http.Request {
				req, _ := http.NewRequest("PRI", "/", bytes.NewBufferString("SM\r\n\r\n"))
				return req
			}(),
			expected: false,
		},
		{
			name: "has wrong proto",
			req: func() *http.Request {
				req, _ := http.NewRequest("PRI", "*", bytes.NewBufferString("SM\r\n\r\n"))
				req.Proto = "HTTP/1.1"

				return req
			}(),
			expected: false,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, server.IsH2C(tc.req))
		})
	}
}
