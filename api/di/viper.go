// Package di provides utilities to help resolve dependencies into valid objects
package di

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/cfg"
	"google.golang.org/grpc"
)

// ErrDependencyFailure means that, for some reason, the dependency required didn't work
var ErrDependencyFailure = errors.New("dependency failure")

// OptsFromViper reads the configuration from viper, and returns options that can bootstrap a gRPC server
func OptsFromViper() ([]grpc.ServerOption, error) {
	opts := []grpc.ServerOption{}

	// The interceptor is a soft dependency — it can fail. Here, we're indicating that failure through the
	// cfg.ErrMissingOptions
	icept, err := jwts.WireServerInterceptor()
	if err != nil && !errors.Is(err, cfg.ErrMissingOptions) {
		return nil, fmt.Errorf("%w: %s", ErrDependencyFailure, err)
	} else if err == nil {
		opts = append(
			opts,
			grpc.StreamInterceptor(icept.StreamServerInterceptor),
			grpc.UnaryInterceptor(icept.UnaryServerInterceptor),
		)
	}

	return opts, nil
}
