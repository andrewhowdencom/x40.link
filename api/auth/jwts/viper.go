package jwts

import (
	"github.com/andrewhowdencom/x40.link/api"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/golang-jwt/jwt/v5"
)

// ServerInterceptorOptsFromViper resolves the global viper configuration into a series of options that can
// bootstrap a server interceptor
func ServerInterceptorOptsFromViper() ([]ServerInterceptorOptionFunc, error) {
	if cfg.AuthX40.Value() {
		return []ServerInterceptorOptionFunc{
			WithJWKSKeyFunc(PublicJWKSURL),
			WithParser(jwt.NewParser(PublicJWTClaims...)),
			WithAddedPermissions(api.X40Permissions()),
			WithAddedPermissions(api.ReflectionPermissions()),
		}, nil
	}

	opts := []ServerInterceptorOptionFunc{}
	parserOpts := []jwt.ParserOption{}

	if v := cfg.AuthJWKSURL.Value(); v != "" {
		opts = append(opts, WithJWKSKeyFunc(v))
	}

	if v := cfg.AuthClaimAudience.Value(); v != "" {
		parserOpts = append(parserOpts, jwt.WithAudience(v))
	}

	if v := cfg.AuthClaimIssuer.Value(); v != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(v))
	}

	if cfg.AuthClaimIssuedAt.Value() {
		parserOpts = append(parserOpts, jwt.WithIssuedAt())
	}

	if cfg.AuthClaimExpiration.Value() {
		parserOpts = append(parserOpts, jwt.WithExpirationRequired())
	}

	if len(parserOpts) > 0 {
		opts = append(opts, WithParser(jwt.NewParser(parserOpts...)))
	}

	if len(opts) > 0 {
		return opts, nil
	}

	return nil, cfg.ErrMissingOptions
}
