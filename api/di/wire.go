//go:build wireinject

package di

import (
	"context"

	"github.com/andrewhowdencom/x40.link/api"
	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/grpcserver"
	str "github.com/andrewhowdencom/x40.link/storage/di"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

var GRPCServerSet = wire.NewSet(
	grpcserver.NewGRPCServer,
	grpcserver.NewHealthService,

	api.TracerProviderSet,
	str.StorageSet,

	jwts.NewManager,
	wire.FieldsOf(new(cfg.JWTS), "PrivateKey", "PublicKey"),

	ProvideGRPCOptions,
)

// WireGRPCServer creates a new GRPC Server
func WireGRPCServer(context.Context, cfg.Auth, cfg.JWTS, cfg.Storage) (*grpc.Server, error) {
	wire.Build(GRPCServerSet)
	return &grpc.Server{}, nil
}
