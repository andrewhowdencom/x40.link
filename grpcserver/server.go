package grpcserver

import (
	"context"

	"github.com/andrewhowdencom/x40.link/api/auth"
	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/api/gen/health"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// NewGRPCServer generates a valid GRPC server with all GRPC routes configured.
func NewGRPCServer(storage storage.Storer, authorizer auth.Authorizer, health health.Service, tracer *sdktrace.TracerProvider) (*grpc.Server, error) {
	// Setup the panic handler
	panicHandler := func(p any) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(panicHandler)),
			otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(tracer)),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(panicHandler)),
			otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(tracer)),
		),
	}

	m := grpc.NewServer(opts...)

	// Register the services
	gendev.RegisterManageURLsServer(m, &dev.URL{
		Storer: storage,
	})
	genhealth.RegisterHealthServer(m, health)
	reflection.Register(m)

	return m, nil
}

// NewHealthService creates a new health service
func NewHealthService(ctx context.Context) (health.Service, error) {
	return health.New(), nil
}
