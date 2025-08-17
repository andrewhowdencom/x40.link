// Package auth provides facilities to generate the auth required for connecting to the API
package auth

import (
	"context"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/x40.link/api"
	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/andrewhowdencom/x40.link/api/auth/tokens"
	"github.com/andrewhowdencom/x40.link/api/auth/tokens/seeds"
	"github.com/andrewhowdencom/x40.link/api/auth/tokens/storage"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// TokenSource returns a TokenSource appropriate for the CLI Application, or an error if this failed.
func TokenSource() (oauth2.TokenSource, error) {
	ctx := context.Background()

	cfg := &oauth2.Config{
		ClientID: viper.GetString(cfg.OAuth2ClientID.Path),
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: viper.GetString(cfg.OAuth2DeviceAuthorizationEndpoint.Path),
			TokenURL:      viper.GetString(cfg.OAuth2TokenURL.Path),
		},
		Scopes: api.X40PermissionsList(),
	}

	tokPath, err := xdg.DataFile(filepath.Join("x40", "cli-token"))
	if err != nil {
		return nil, err
	}

	ts, err := tokens.NewCachingSource(ctx, cfg.TokenSource, seeds.DeviceAuth(jwts.AudienceX40API, cfg), &storage.File{
		Path: tokPath,
	})

	if err != nil {
		return nil, err
	}

	return ts, nil
}
