//go:build wireinject

package di

import (
	"github.com/andrewhowdencom/x40.link/api"
	str "github.com/andrewhowdencom/x40.link/storage/di"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

func WireGRPCServer() (*grpc.Server, error) {
	wire.Build(api.NewGRPCMux, OptsFromViper, str.WireStorage)

	return &grpc.Server{}, nil
}
