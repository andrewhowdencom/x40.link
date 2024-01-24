package jwts_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/api/auth"
	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestWithAddedRoles(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  []jwts.ServerInterceptorOptionFunc

		expected map[string]string
	}{
		{
			name:     "empty map",
			opt:      []jwts.ServerInterceptorOptionFunc{jwts.WithAddedPermissions(map[string]string{})},
			expected: map[string]string{},
		},
		{
			name: "single add",
			opt: []jwts.ServerInterceptorOptionFunc{
				jwts.WithAddedPermissions(map[string]string{
					"foo": "bar",
				}),
			},
			expected: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "with override",
			opt: []jwts.ServerInterceptorOptionFunc{
				jwts.WithAddedPermissions(map[string]string{
					"foo": "bar",
				}),
				jwts.WithAddedPermissions(map[string]string{
					"foo": "baz",
				}),
			},
			expected: map[string]string{
				"foo": "baz",
			},
		},
		{
			name: "multiple add",
			opt: []jwts.ServerInterceptorOptionFunc{
				jwts.WithAddedPermissions(map[string]string{
					"foo": "bar",
				}),
				jwts.WithAddedPermissions(map[string]string{
					"bar": "baz",
				}),
			},
			expected: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			o, err := jwts.NewServerInterceptor(tc.opt...)
			assert.Nil(t, err)

			assert.Equal(t, tc.expected, o.Permissions)
		})
	}
}

// TestJWTValidation validates that the incoming context carries an appropriate authorization header, with roles
// suitable for this application.
func TestJWTValidation(t *testing.T) {
	t.Parallel()

	// Generate key material
	tk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic("failed to generate private key: " + err.Error())
	}

	// test claims
	tClaim := jwt.MapClaims{
		// 			// All Fine
		"iss": "https://issuer.local",
		"sub": "e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d",
		"aud": "https://api.x40.local",

		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour * 4)),

		jwts.ClaimPermissions: []string{
			"TEST-METHOD-PERMISSION",
		},
	}

	tParser := jwt.NewParser(
		jwt.WithIssuer("https://issuer.local"),
		jwt.WithAudience("https://api.x40.local"),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	// TODO: Test sub
	for _, tc := range []struct {
		name string
		opts []jwts.ServerInterceptorOptionFunc

		ctx    context.Context
		method string

		err    error
		retCtx context.Context
	}{
		{
			name: "no roles defined",
			ctx:  context.Background(),

			method: "TEST-METHOD-NAME",

			err:    auth.ErrCannotAuthorize,
			retCtx: context.Background(),
		},
		{
			name: "missing metadata",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(tk),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx:    context.Background(),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrMissingMetadata,
			retCtx: context.Background(),
		},
		{
			name: "missing authorization key",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(tk),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx:    metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{})),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrMissingAuthorization,
			retCtx: context.Background(),
		},
		{
			name: "corrupted authorization key",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(tk),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					auth.MetaKeyAuthorization, "first-item",
					auth.MetaKeyAuthorization, "second-item",
				),
			),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrCorruptedAuthorization,
			retCtx: context.Background(),
		},
		{
			name: "token signed by wrong key",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(func() *rsa.PublicKey {
					// Generate key material
					tk, err := rsa.GenerateKey(rand.Reader, 1024)
					if err != nil {
						panic("failed to generate private key: " + err.Error())
					}

					return &tk.PublicKey
				}()),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},
			ctx: func() context.Context {
				vc := tClaim
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, vc)

				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to prepare test token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "no role",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(&tk.PublicKey),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx: func() context.Context {
				claims := jwt.MapClaims{}

				for k, v := range tClaim {
					if k == jwts.ClaimPermissions {
						continue
					}

					claims[k] = v
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to prepare test token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "wrong audience",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(&tk.PublicKey),
				jwts.WithParser(tParser),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx: func() context.Context {
				claims := jwt.MapClaims{}

				for k, v := range tClaim {
					claims[k] = v
				}

				claims["aud"] = "https://another-api.x40.local"

				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to prepare test token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "no subject",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(&tk.PublicKey),
				jwts.WithParser(tParser),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx: func() context.Context {
				claims := jwt.MapClaims{}

				for k, v := range tClaim {
					if k == "sub" {
						continue
					}

					claims[k] = v
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to prepare test token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-METHOD-NAME",

			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		// There are methods for which we want to allow anonymous access. One example is the "reflection API"
		//
		// See https://github.com/grpc/grpc/blob/master/doc/server-reflection.md
		{
			name: "no permissions required",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(&tk.PublicKey),
				jwts.WithParser(tParser),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-ANONYMOUS-METHOD": "",
				}),
			},

			ctx: func() context.Context {
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, tClaim)

				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to prepare test token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-ANONYMOUS-METHOD",

			err:    nil,
			retCtx: context.WithValue(context.Background(), storage.CtxKeyAgent, "sub:e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d"),
		},
		{
			name: "all ok, has agent context",
			opts: []jwts.ServerInterceptorOptionFunc{
				jwts.WithStaticKey(&tk.PublicKey),
				jwts.WithParser(tParser),
				jwts.WithAddedPermissions(map[string]string{
					"TEST-METHOD-NAME": "TEST-METHOD-PERMISSION",
				}),
			},

			ctx: func() context.Context {
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, tClaim)
				sTok, err := tok.SignedString(tk)
				if err != nil {
					panic("failed to create token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			method: "TEST-METHOD-NAME",

			err:    nil,
			retCtx: context.WithValue(context.Background(), storage.CtxKeyAgent, "sub:e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d"),
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			jwt, err := jwts.NewServerInterceptor(tc.opts...)

			if err != nil {
				t.Errorf("construction failed: %s", err)
			}

			ctx, err := jwt.ValidateCtx(tc.ctx, tc.method)

			// Ensure the agent is what we expect
			assert.Equal(t, tc.retCtx.Value(storage.CtxKeyAgent), ctx.Value(storage.CtxKeyAgent))

			// Ensure that, where it fails, it is failing as expected.
			assert.ErrorIs(t, err, tc.err)

			// Check that the authentication header is stripped out
			md, _ := metadata.FromIncomingContext(ctx)
			assert.Nil(t, md.Get(auth.MetaKeyAuthorization))
		})
	}
}
