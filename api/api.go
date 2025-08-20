// Package api bootstraps and inits the API stubs connecting the server with the concrete implementations.
package api

import (
	"context"
	"fmt"

	"github.com/andrewhowdencom/x40.link/api/auth"
	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/api/gen/health"
	"github.com/andrewhowdencom/x40.link/api/gen/health"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)


// NewGRPCClient generates a client able to talk gRPC to the API
func NewGRPCClient(ctx context.Context, addr string, opts ...grpc.DialOption) (gendev.ManageURLsClient, error) {
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to grpc server: %w", err)
	}

	return gendev.NewManageURLsClient(conn), nil
}
