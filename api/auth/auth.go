// Package auth provides authentication implementation(s) that can be used to limit access to the (gRPC) server.
package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
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

// OIDC provides an implementation for the OpenID method of verifying users.
//
// Only implements authentication (or "authn" — "who are you")
//
// TODO: Implement stream interface
type OIDC struct {
	Verifier     *oidc.IDTokenVerifier
	NeededClaims map[string]map[string]interface{}
}

// UnaryServerInterceptor provides the implementation of the OIDC handler
func (o *OIDC) UnaryServerInterceptor(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	isAuthenticated, err := o.authn(ctx)

	if !isAuthenticated {
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
	isAuthenticated, err := o.authn(context.Background())

	if !isAuthenticated {
		return err
	}

	return handler(srv, ss)
}

// authn is the implementation of whether or not a given user is allowed to access a resource. Indicates
// this by returning a bool, and if not allowed, an error describing why.
func (o *OIDC) authn(ctx context.Context) (bool, error) {
	m, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false, ErrMissingMetadata
	}

	v, ok := m[MetaKeyAuthorization]
	if !ok {
		return false, ErrMissingAuthorization
	}

	if len(v) != 1 {
		return false, ErrCorruptedAuthorization
	}

	// TODO: Should be B|b
	strTok, _ := strings.CutPrefix(v[0], "Bearer ")

	tok, err := o.Verifier.Verify(ctx, strTok)

	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrFailedToAuthenticate, err)
	}

	inClaims := make(map[string]interface{}, 0)
	if err := tok.Claims(&inClaims); err != nil {
		return false, fmt.Errorf("%w: %s", ErrFailedToAuthenticate, err)
	}

	return true, nil
}
