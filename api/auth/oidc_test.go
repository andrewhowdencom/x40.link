package auth_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/api/auth"
	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

// TestOIDCAuthCtx validates that the incoming context carries an appropriate authorization header, with roles
// suitable for this application.
//
// Uses a stubbed OIDC Verifier. See:
// 1. https://golang-jwt.github.io/jwt/usage/create/
func TestOIDCAuthCtx(t *testing.T) {
	t.Parallel()

	clientID, issuer := "i-am-the-client-id", "i-am-the-issuer"

	// Generate key material
	pkey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic("failed to generate private key: " + err.Error())
	}

	ks := &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{pkey.Public()}}
	verifier := oidc.NewVerifier(issuer, ks, &oidc.Config{
		ClientID:             clientID,
		SupportedSigningAlgs: []string{oidc.RS256},
	})

	for _, tc := range []struct {
		name string
		ctx  context.Context

		err    error
		retCtx context.Context
	}{
		{
			name: "missing metadata",
			ctx:  context.Background(),

			err:    auth.ErrMissingMetadata,
			retCtx: context.Background(),
		},
		{
			name: "missing authorization key",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{})),

			err:    auth.ErrMissingAuthorization,
			retCtx: context.Background(),
		},
		{
			name: "corrupted authorization key",
			ctx: metadata.NewIncomingContext(context.
				Background(),
				metadata.Pairs(
					auth.MetaKeyAuthorization, "first-item",
					auth.MetaKeyAuthorization, "second-item",
				),
			),

			err:    auth.ErrCorruptedAuthorization,
			retCtx: context.Background(),
		},
		{
			name: "invalid token",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				auth.MetaKeyAuthorization: "🙉🙈🙊",
			})),

			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "no role",
			ctx: func() context.Context {
				oTok := &jwts.OIDC{
					Default: &jwts.Default{
						Issuer:  issuer,
						Subject: "e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d",

						Audience: clientID,

						IssuedAt:   jwt.NewNumericDate(time.Now()),
						NotBefore:  jwt.NewNumericDate(time.Now()),
						Expiration: jwt.NewNumericDate(time.Now().Add(time.Hour * 4)),
					},

					Name:   "Test User",
					Email:  "test-user@example.local",
					Locale: "en_GB",
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, oTok)
				sTok, err := tok.SignedString(pkey) //nolint:all false positive
				if err != nil {
					panic("failed to create token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "corrupt claims",
			ctx: func() context.Context {
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					// All Fine
					"iss": issuer,
					"sub": "e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d",
					"aud": clientID,

					"iat": jwt.NewNumericDate(time.Now()),
					"nbf": jwt.NewNumericDate(time.Now()),
					"exp": jwt.NewNumericDate(time.Now().Add(time.Hour * 4)),

					// Corrupt
					auth.RoleNamespace: "i'm not an array",
				})

				sTok, err := tok.SignedString(pkey)
				if err != nil {
					panic("failed to create token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),
			err:    auth.ErrFailedToAuthenticate,
			retCtx: context.Background(),
		},
		{
			name: "all ok, has agent context",
			ctx: func() context.Context {
				oTok := &jwts.X40{
					Default: &jwts.Default{
						Issuer:  issuer,
						Subject: "e7e90d06-b60b-11ee-993a-5bf4ddaa2f8d",

						Audience: clientID,

						IssuedAt:   jwt.NewNumericDate(time.Now()),
						NotBefore:  jwt.NewNumericDate(time.Now()),
						Expiration: jwt.NewNumericDate(time.Now().Add(time.Hour * 4)),
					},

					OIDC: &jwts.OIDC{
						Name:   "Test User",
						Email:  "test-user@example.local",
						Locale: "en_GB",
					},

					// Custom
					Roles: []string{
						auth.RoleAPIUser,
					},
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, oTok)
				sTok, err := tok.SignedString(pkey)
				if err != nil {
					panic("failed to create token: " + err.Error())
				}

				return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
					auth.MetaKeyAuthorization: sTok,
				}))
			}(),

			err:    nil,
			retCtx: context.WithValue(context.Background(), storage.CtxKeyAgent, "email:test-user@example.local"),
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			oidc := &auth.OIDC{
				Verifier: verifier,
			}

			ctx, err := oidc.VerifyCtx(tc.ctx)
			assert.Equal(t, tc.retCtx.Value(storage.CtxKeyAgent), ctx.Value(storage.CtxKeyAgent))
			assert.ErrorIs(t, err, tc.err)

			// Check that the authentication header is stripped out
			md, _ := metadata.FromIncomingContext(ctx)
			assert.Nil(t, md.Get(auth.MetaKeyAuthorization))
		})
	}

	assert.Nil(t, err)
}
