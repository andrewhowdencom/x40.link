// Package api bootstraps and configures the API stubs connecting the server with the concrete, business logic
// implementations.
package api

import (
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"github.com/andrewhowdencom/x40.link/api/dev"
	gendev "github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/uid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	descpb "google.golang.org/protobuf/types/descriptorpb"
)

// Err* are common error codes
var (
	ErrCannotDialServer    = errors.New("cannot connect to grpc server")
	ErrMissingCertificates = errors.New("cannot get system certificates")
)

// Client is the common interface for the gRPC client
type Client interface {
	gendev.ManageURLsClient
}

// ProtoPackages is a list of all protobuf packages this API cares about.
var ProtoPackages = []string{
	"x40.dev.url",
	"x40.dev.auth",
}

// ReflectionPermissions are permissions from the reflection API.
//
// See
func ReflectionPermissions() map[string]string {
	return map[string]string{
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo": "",
	}
}

// X40Permissions returns a paired list of method + scope definitions.
func X40Permissions() map[string]string {
	ret := map[string]string{}

	for _, p := range ProtoPackages {
		protoregistry.GlobalFiles.RangeFilesByPackage(protoreflect.FullName(p), func(fd protoreflect.FileDescriptor) bool {
			for i := 0; i < fd.Services().Len(); i++ {
				svc := fd.Services().Get(i)

				for j := 0; j < svc.Methods().Len(); j++ {
					method := svc.Methods().Get(j)

					opts := method.Options().(*descpb.MethodOptions)
					scope := proto.GetExtension(opts, gendev.E_Oauth2Scope).(string)

					// Here, we need to construct the name as the interceptor returns it. That is,
					// /<package>/<method>
					name := strings.Join([]string{
						"/",
						string(svc.FullName()),
						"/",
						string(method.Name()),
					}, "")

					ret[name] = scope

				}
			}

			return true
		})
	}

	return ret
}

// X40PermissionsList returns a list of all permissions, but no method association.
func X40PermissionsList() []string {
	m := []string{}

	for _, v := range X40Permissions() {
		m = append(m, v)
	}

	return m
}

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

// NewGRPCClient generates a client able to talk gRPC to the API
func NewGRPCClient(addr string, opts ...grpc.DialOption) (Client, error) {

	// Use the default system certiifcate pool.
	cp, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrMissingCertificates, err)
	}

	opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cp, "")))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotDialServer, err)
	}

	return gendev.NewManageURLsClient(conn), nil
}
