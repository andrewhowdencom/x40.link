//go:build wireinject

package server

import (
	"errors"
	"fmt"
	"net/http"

	apidi "github.com/andrewhowdencom/x40.link/api/di"
	"github.com/andrewhowdencom/x40.link/cfg"
	strdi "github.com/andrewhowdencom/x40.link/storage/di"
	"github.com/google/wire"
)

// ErrDependencyFailure just means there was a failure resolving a dependency
var ErrDependencyFailure = errors.New("dependency failure")

// ResolveOptions generates a server with the appropriate configuration, based on Viper and other
// required dependencies
func ResolveOptions() ([]Option, error) {
	opts := []Option{}

	storage, err := strdi.WireStorage()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDependencyFailure, err)
	}

	opts = append(opts, WithStorage(storage))

	if cfg.ServerH2CEnabled.Value() {
		opts = append(opts, WithH2C())
	}

	server, err := apidi.WireGRPCServer()
	if err != nil && !errors.Is(err, cfg.ErrMissingOptions) {
		return nil, ErrDependencyFailure
	} else if err == nil {
		opts = append(opts, WithGRPC(cfg.ServerAPIGRPCHost.Value(), server))
	}

	return opts, nil
}

func WireServer() (*http.Server, error) {
	wire.Build(New, ResolveOptions)
	return &http.Server{}, nil
}
