// Package api bootstraps and configures the API stubs connecting the server with the concrete, business logic
// implementations.
package api

import (
	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewGRPCMux generates a valid GRPC server with all GRPC routes configured.
func NewGRPCMux(opts ...grpc.ServerOption) *grpc.Server {
	m := grpc.NewServer(opts...)

	dev.RegisterManageURLsServer(m, &dev.UnimplementedManageURLsServer{})

	reflection.Register(m)

	return m
}
