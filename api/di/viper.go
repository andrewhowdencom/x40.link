// Package di provides utilities to help resolve dependencies into valid objects
package di

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/cfg"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// ErrDependencyFailure means that, for some reason, the dependency required didn't work
var ErrDependencyFailure = errors.New("dependency failure")

// OptsFromViper reads the configuration from viper, and returns options that can bootstrap a gRPC server
func OptsFromViper() ([]grpc.ServerOption, error) {
	opts := []grpc.ServerOption{}

	// otelgrpc emits the parent span per RPC and provides server-side
	// metrics. We always include these interceptors — when otel.Init
	// hasn't been called, the OTel SDK falls back to a no-op tracer /
	// meter provider, so the cost is negligible and the wiring remains
	// unconditional across configurations.
	opts = append(opts,
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

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
