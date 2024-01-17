// Package api bootstraps and configures the API stubs connecting the server with the concrete, business logic
// implementations.
package api

import (
	"github.com/andrewhowdencom/x40.link/api/dev"
	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/uid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewGRPCMux generates a valid GRPC server with all GRPC routes configured.
func NewGRPCMux(storer storage.Storer, opts ...grpc.ServerOption) *grpc.Server {
	m := grpc.NewServer(opts...)

	gendev.RegisterManageURLsServer(m, &dev.URL{
		Storer: storer,
		Enricher: (&dev.URLEnricher{
			Domain: "x40.link",
			Path:   uid.New(uid.TypeRandom),
		}).Enrich,
	})

	reflection.Register(m)

	return m
}
