// Package api bootstraps and configures the API stubs connecting the server with the concrete, business logic
// implementations.
package api

import (
	"context"

	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewGRPCGatewayMux provides a mux with all GRPC Gateway routes configured.
func NewGRPCGatewayMux() *runtime.ServeMux {

	m := runtime.NewServeMux()

	// Register the gateway routes
	if err := dev.RegisterManageURLsHandlerServer(context.Background(), m, &dev.UnimplementedManageURLsServer{}); err != nil {
		panic(err)
	}

	return m
}

// NewGRPCMux generates a valid GRPC server with all GRPC routes configured.
func NewGRPCMux() *grpc.Server {
	m := grpc.NewServer()

	dev.RegisterManageURLsServer(m, &dev.UnimplementedManageURLsServer{})

	reflection.Register(m)

	return m
}
