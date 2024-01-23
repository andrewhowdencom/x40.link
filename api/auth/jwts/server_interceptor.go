package jwts

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/andrewhowdencom/x40.link/api/auth"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/golang-jwt/jwt/v5"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Err* are sentinel errors.
var (
	ErrOpt = errors.New("unable to apply option")
)

var (
	reBearer = regexp.MustCompile("(?i)Bearer ")
)

// Public* is the configuration for the public endpoints.
var (
	PublicJWKSURL = "https://x40.eu.auth0.com/.well-known/jwks.json"

	// By default, quite a few of the JWT Claims are optional. However, we want them to be, by default, active.
	// Here, we configure the claims as we expected.
	//
	// See:
	// 1. https://auth0.com/docs/secure/tokens/json-web-tokens/json-web-token-claims#registered-claims
	// 2.  https://auth0.com/docs/secure/tokens/access-tokens/get-access-tokens
	// 3. https://auth0.com/docs/secure/tokens/token-best-practices
	PublicJWTClaims = []jwt.ParserOption{
		// The issuer of the domain that we request tokens from.
		jwt.WithIssuer("https://x40.eu.auth0.com/"),

		// The audience is the auth0 resource server (or API)
		jwt.WithAudience("https://api.x40.link"),

		// Time based claims
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	}
)

// ServerOptionFunc modifies the behavior of the oauth2 validator
type ServerOptionFunc func(o *ServerInterceptor) error

// ServerInterceptor is an interceptor that validates the JWT tokens supplied by the user.
// See:
// 1. https://auth0.com/docs/secure/tokens/access-tokens/validate-access-tokens
type ServerInterceptor struct {
	// Permissions are the scopes that a given user is expected to have for the supplied method.
	Permissions map[string]string

	// kf allow supplying the key func, so that it can be overridden in tests, or so that a
	// JWKS endpoint can be used.
	kf  jwt.Keyfunc
	par *jwt.Parser
}

// WithKeyFunc supplies the function that supplies the key for validation
func WithKeyFunc(kf jwt.Keyfunc) ServerOptionFunc {
	return func(o *ServerInterceptor) error {
		o.kf = kf

		return nil
	}
}

// WithJWKSKeyFunc allows fetching the key function from upstream
func WithJWKSKeyFunc(urls ...string) ServerOptionFunc {
	return func(o *ServerInterceptor) error {
		n, err := keyfunc.NewDefault(urls)
		if err != nil {
			return err
		}

		o.kf = n.Keyfunc

		return nil
	}
}

// WithStaticKey allows using an arbitrary static key to check for the token validity. WARNING: Should not really be
// used; primarily designed for ease of testing.
func WithStaticKey(k interface{}) ServerOptionFunc {
	return func(o *ServerInterceptor) error {
		o.kf = func(t *jwt.Token) (interface{}, error) {
			return k, nil
		}

		return nil
	}
}

// WithAddedPermissions sets the scopes directly on the oauth2 implementation.
// TODO: Test this.
func WithAddedPermissions(perms map[string]string) ServerOptionFunc {
	return func(o *ServerInterceptor) error {
		for k, v := range perms {
			o.Permissions[k] = v
		}

		return nil
	}
}

// WithParser allows configuring the parser.
func WithParser(p *jwt.Parser) ServerOptionFunc {
	return func(o *ServerInterceptor) error {
		o.par = p
		return nil
	}
}

// NewValidator is a convenience function that generates the JWT validation interceptors
func NewValidator(opts ...ServerOptionFunc) (*ServerInterceptor, error) {
	o := &ServerInterceptor{
		Permissions: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrOpt, err)
		}
	}

	// Ensure that, by default, the production key validation is available.
	if o.kf == nil {
		if err := WithJWKSKeyFunc(PublicJWKSURL)(o); err != nil {
			return nil, err
		}
	}

	// Ensure that, by default, the production parser configuration is available
	if o.par == nil {
		o.par = jwt.NewParser(PublicJWTClaims...)
	}

	return o, nil
}

// UnaryServerInterceptor provides the implementation of the OIDC Verifier
func (o *ServerInterceptor) UnaryServerInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {

	var err error
	ctx, err = o.ValidateCtx(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// ValidateCtx is a shared function that validates the metadata associated with this request has the required token,
// and that the token has the expected permissions.
func (o *ServerInterceptor) ValidateCtx(
	ctx context.Context,
	method string,
) (context.Context, error) {
	_, ok := o.Permissions[method]
	if !ok {
		return ctx, fmt.Errorf("%w: %s (%s)", auth.ErrCannotAuthorize, "no scope for the method", method)
	}

	m, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return ctx, auth.ErrMissingMetadata
	}

	inTok, ok := m[auth.MetaKeyAuthorization]
	if !ok {
		return ctx, auth.ErrMissingAuthorization
	}

	m.Delete(auth.MetaKeyAuthorization)
	ctx = metadata.NewIncomingContext(ctx, m)

	if len(inTok) != 1 {
		return ctx, auth.ErrCorruptedAuthorization
	}

	// Strip bearer
	strTok := reBearer.ReplaceAllString(inTok[0], "")

	claims := &X40{
		Needs: NeedsPermission(o.Permissions[method]),
	}

	_, err := o.par.ParseWithClaims(strTok, claims, o.kf)
	if err != nil {
		return ctx, fmt.Errorf("%w: %s", auth.ErrFailedToAuthenticate, err)
	}

	ctx = context.WithValue(ctx, storage.CtxKeyAgent, "sub:"+claims.Subject)

	return ctx, nil
}

// StreamServerInterceptor provides the implementation of the OIDC Verifier
func (o *ServerInterceptor) StreamServerInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	ctx, err := o.ValidateCtx(ss.Context(), info.FullMethod)

	if err != nil {
		return err
	}

	wrappedStream := middleware.WrapServerStream(ss)
	wrappedStream.WrappedContext = ctx

	return handler(srv, wrappedStream)
}
