// Package auth provides facilities to generate the auth required for connecting to the API
package auth

import (
	"context"

	"golang.org/x/oauth2"
)

// perRPCCredentials is an implementation of the PerRPCCredentials interface, allowing each
// gRPC call to have the required Authorization header injected.
type perRPCCredentials struct {
	ts oauth2.TokenSource
}

// GetRequestMetadata implements the PerRPCCredentials interface, allowing the user to get the required
// request metadata.
func (p *perRPCCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	tok, err := p.ts.Token()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"Authorization": "Bearer " + tok.AccessToken,
	}, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface, allowing the user to determine
// whether transport security is required.
func (p *perRPCCredentials) RequireTransportSecurity() bool {
	return true
}

// NewPerRPCCredentials creates a new PerRPCCredentials that can be used with a gRPC client
func NewPerRPCCredentials(ts oauth2.TokenSource) *perRPCCredentials {
	return &perRPCCredentials{
		ts: ts,
	}
}
