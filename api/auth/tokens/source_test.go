package tokens_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/api/auth/tokens"
	"github.com/andrewhowdencom/x40.link/api/auth/tokens/seeds"
	"github.com/andrewhowdencom/x40.link/api/auth/tokens/storage"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var (
	// ErrSentinel is just to read and write hte same error
	ErrSentinel = errors.New("a sentinel error for testing")
)

type TokenSource struct {
	tokens []*oauth2.Token

	err error
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	if ts.err != nil {
		return nil, ts.err
	}

	if len(ts.tokens) == 0 {
		return nil, ErrSentinel
	}

	var next *oauth2.Token
	next, ts.tokens = ts.tokens[len(ts.tokens)-1], ts.tokens[:len(ts.tokens)]

	return next, nil
}

func TestNewCachingSource(t *testing.T) {
	// s* are sentinels, generated and used ot validate
	sExpiry := time.Now().Add(time.Hour * 60)
	sAccessToken := "I-AM-THE-ACCESS-TOKEN"
	sRefreshToken := "I-AM-THE-REFRESH-TOKEN"

	for _, tc := range []struct {
		name string

		ctx context.Context
		cfg *oauth2.Config
		sf  seeds.Seed
		str storage.Storage

		err error
		tok *oauth2.Token
	}{
		{
			name: "storage fails",
			ctx:  context.Background(),
			cfg:  &oauth2.Config{},
			sf: func(_ context.Context) (*oauth2.Token, error) {
				return nil, fmt.Errorf("doesnt matter")
			},

			str: storage.NewTest(storage.WithReadError(func(_ *storage.Test) error {
				return storage.ErrStorageFailure
			})),

			err: tokens.ErrCacheSetupFailed,
		},
		{
			name: "no token, seed fails",
			ctx:  context.Background(),
			cfg:  &oauth2.Config{},
			sf: func(_ context.Context) (*oauth2.Token, error) {
				return nil, seeds.ErrFailed
			},

			str: storage.NewTest(),

			err: tokens.ErrCacheSetupFailed,
		},
		{
			name: "no token, seed fails to write to storage",
			ctx:  context.Background(),
			cfg:  &oauth2.Config{},
			sf: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken:  sAccessToken,
					RefreshToken: sRefreshToken,
					TokenType:    "Bearer",
					Expiry:       time.Now().Add(time.Hour),
				}, nil
			},
			str: storage.NewTest(storage.WithWriteError(func(_ *storage.Test, _ []byte) error {
				return storage.ErrStorageFailure
			})),

			err: tokens.ErrCacheSetupFailed,
		},
		{
			name: "junk returned from storage",
			ctx:  context.Background(),
			cfg:  &oauth2.Config{},
			sf: func(_ context.Context) (*oauth2.Token, error) {
				return nil, fmt.Errorf("doesnt matter")
			},
			str: storage.NewTest(storage.WithBytes([]byte("Whoops Im not JSON!"))),

			err: tokens.ErrCacheFailure,
		},
		{
			name: "expired token from storage",
			ctx:  context.Background(),
			cfg:  &oauth2.Config{},
			sf: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken:  "I-AM-THE-ACCESS-TOKEN",
					RefreshToken: "I-AM-THE-REFRESH-TOKEN",
					TokenType:    "Bearer",
					Expiry:       sExpiry,
				}, nil
			},

			str: storage.NewTest(storage.WithBytes([]byte(`
{
	"access_token": "I-AM-AN-ACCESS-TOKEN",
	"token_type": "Bearer",
	"refresh_token": "",
	"expiry": "2024-01-25T16:37:22.176803588+01:00"
}
			`))),

			err: nil,
			tok: &oauth2.Token{
				AccessToken:  "I-AM-THE-ACCESS-TOKEN",
				RefreshToken: "I-AM-THE-REFRESH-TOKEN",
				TokenType:    "Bearer",
				Expiry:       sExpiry,
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts, err := tokens.NewCachingSource(tc.ctx, tc.cfg.TokenSource, tc.sf, tc.str)

			assert.ErrorIs(t, err, tc.err)

			if tc.tok != nil {
				nt, _ := ts.Token()
				assert.Equal(t, tc.tok, nt)
			}
		})
	}
}

func TestTokenSource(t *testing.T) {
	// s* are sentinels, generated and used ot validate
	sExpiry := time.Now().Add(time.Hour * 60)
	sAccessToken := "I-AM-THE-ACCESS-TOKEN"
	sRefreshToken := "I-AM-THE-REFRESH-TOKEN"

	for _, tc := range []struct {
		name string
		str  storage.Storage
		tsf  func(ctx context.Context, t *oauth2.Token) oauth2.TokenSource
		seed seeds.Seed

		tok *oauth2.Token
		err error
	}{
		{
			name: "underlying token source failure",
			str:  storage.NewTest(),

			tsf: func(_ context.Context, _ *oauth2.Token) oauth2.TokenSource {
				return &TokenSource{
					err: ErrSentinel,
				}
			},
			seed: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken:  sAccessToken,
					RefreshToken: sRefreshToken,
					TokenType:    "Bearer",
					Expiry:       sExpiry,
				}, nil
			},

			err: ErrSentinel,
		},
		{
			name: "provided token is exactly the same",
			str:  storage.NewTest(),
			seed: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken:  sAccessToken,
					RefreshToken: sRefreshToken,
					TokenType:    "Bearer",
					Expiry:       sExpiry,
				}, nil
			},
			tsf: func(_ context.Context, _ *oauth2.Token) oauth2.TokenSource {
				return &TokenSource{
					tokens: []*oauth2.Token{
						{
							AccessToken:  sAccessToken,
							RefreshToken: sRefreshToken,
							TokenType:    "Bearer",
							Expiry:       sExpiry,
						},
					},
				}
			},

			tok: &oauth2.Token{
				AccessToken:  sAccessToken,
				RefreshToken: sRefreshToken,
				TokenType:    "Bearer",
				Expiry:       sExpiry,
			},
		},
		{
			name: "token different, but storage failed",
			str: storage.NewTest(storage.WithWriteError(func(_ *storage.Test, b []byte) error {
				// The initial write is empty, and contains only the JSON and metadata. Given this, we skip the error
				// if this is teh write.
				if len(b) < 60 {
					return nil
				}

				return ErrSentinel
			})),
			seed: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{}, nil
			},
			tsf: func(_ context.Context, _ *oauth2.Token) oauth2.TokenSource {
				return &TokenSource{
					tokens: []*oauth2.Token{
						{
							AccessToken:  sAccessToken,
							RefreshToken: sRefreshToken,
							TokenType:    "Bearer",
							Expiry:       sExpiry,
						},
					},
				}
			},
			tok: nil,
			err: ErrSentinel,
		},
		{
			name: "token different, all good",
			str:  storage.NewTest(),
			seed: func(_ context.Context) (*oauth2.Token, error) {
				return &oauth2.Token{}, nil
			},
			tsf: func(_ context.Context, _ *oauth2.Token) oauth2.TokenSource {
				return &TokenSource{
					tokens: []*oauth2.Token{
						{
							AccessToken:  sAccessToken,
							RefreshToken: sRefreshToken,
							TokenType:    "Bearer",
							Expiry:       sExpiry,
						},
					},
				}
			},
			tok: &oauth2.Token{
				AccessToken:  sAccessToken,
				RefreshToken: sRefreshToken,
				TokenType:    "Bearer",
				Expiry:       sExpiry,
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cs, err := tokens.NewCachingSource(
				context.Background(),
				tc.tsf,
				tc.seed,
				tc.str,
			)

			assert.Nil(t, err)

			tok, err := cs.Token()

			assert.Equal(t, tc.tok, tok)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
