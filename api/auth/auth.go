// Package auth provides authentication implementation(s) that can be used to limit access to the (gRPC) server.
// Specific implementations are in each sub package.
package auth

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Meta* are the keys to metadata.
//
// See https://godoc.org/google.golang.org/grpc/metadata#New
const (
	MetaKeyAuthorization = "authorization"
)

// CtxKey is a type just to prevent collisions
type CtxKey string

// CtxKey* are context keys specific to this auth package.
const (
	CtxKeyRoles CtxKey = "roles"
)

// Err* are sentinel errors
var (
	// Server Side
	ErrMissingMetadata        = status.Error(codes.InvalidArgument, "missing metadata")
	ErrMissingAuthorization   = status.Error(codes.InvalidArgument, "missing authorization")
	ErrCorruptedAuthorization = status.Error(codes.InvalidArgument, "unexpected number of authentication values")
	ErrFailedToAuthenticate   = status.Error(codes.Unauthenticated, "unable to authenticate user")
	ErrCannotAuthorize        = status.Error(codes.FailedPrecondition, "cannot authorize message")
)
