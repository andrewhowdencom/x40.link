//go:build wireinject

package jwts

import "github.com/google/wire"

// WireServerInterceptor generates a server interceptor from the global DI container.
func WireServerInterceptor() (*ServerInterceptor, error) {
	wire.Build(NewServerInterceptor, ServerInterceptorOptsFromViper)

	return &ServerInterceptor{}, nil
}
