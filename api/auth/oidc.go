// Package auth provides authentication implementation(s) that can be used to limit access to the (gRPC) server.
package auth

import (
	"context"
	"fmt"
	"regexp"

	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/coreos/go-oidc/v3/oidc"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Meta* are the keys to metadata.
//
// See https://godoc.org/google.golang.org/grpc/metadata#New
const (
	MetaKeyAuthorization = "authorization"
)

// Err* are sentinel errors
var (
	ErrMissingMetadata        = status.Error(codes.InvalidArgument, "missing metadata")
	ErrMissingAuthorization   = status.Error(codes.InvalidArgument, "missing authorization")
	ErrCorruptedAuthorization = status.Error(codes.InvalidArgument, "unexpected number of authentication values")
	ErrFailedToAuthenticate   = status.Error(codes.Unauthenticated, "unable to authenticate user")
)

var (
	reBearer = regexp.MustCompile("(?i)Bearer ")
)

// OIDC provides an implementation for the OpenID method of verifying users.
//
// Only implements authentication (or "authn" — "who are you")
type OIDC struct {
	Verifier *oidc.IDTokenVerifier
}

// UnaryServerInterceptor provides the implementation of the OIDC handler
func (o *OIDC) UnaryServerInterceptor(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	ctx, err := o.authn(ctx)

	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// StreamServerInterceptor provides the implementation of the OIDC handler
func (o *OIDC) StreamServerInterceptor(
	srv any,
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	ctx, err := o.authn(ss.Context())

	if err != nil {
		return err
	}

	wrappedStream := middleware.WrapServerStream(ss)
	wrappedStream.WrappedContext = ctx

	return handler(srv, wrappedStream)
}

// authn is the implementation of whether or not a given user is allowed to access a resource. Indicates
// this by returning a bool, and if not allowed, an error describing why.
func (o *OIDC) authn(ctx context.Context) (context.Context, error) {
	m, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return ctx, ErrMissingMetadata
	}

	v, ok := m[MetaKeyAuthorization]
	if !ok {
		return ctx, ErrMissingAuthorization
	}

	if len(v) != 1 {
		return ctx, ErrCorruptedAuthorization
	}

	// Strip bearer
	strTok := reBearer.ReplaceAllString(v[0], "")

	tok, err := o.Verifier.Verify(ctx, strTok)

	if err != nil {
		return ctx, fmt.Errorf("%w: %s", ErrFailedToAuthenticate, err)
	}

	inClaims := struct {
		Email string `json:"email"`
	}{}

	if err := tok.Claims(&inClaims); err != nil {
		return ctx, fmt.Errorf("%w: %s", ErrFailedToAuthenticate, err)
	}

	return context.WithValue(ctx, storage.CtxKeyAgent, "email:"+inClaims.Email), nil
}
