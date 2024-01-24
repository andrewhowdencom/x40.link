// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package jwts

// Injectors from wire.go:

// WireServerInterceptor generates a server interceptor from the global DI container.
func WireServerInterceptor() (*ServerInterceptor, error) {
	v, err := ServerInterceptorOptsFromViper()
	if err != nil {
		return nil, err
	}
	serverInterceptor, err := NewServerInterceptor(v...)
	if err != nil {
		return nil, err
	}
	return serverInterceptor, nil
}
