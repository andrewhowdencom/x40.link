package seeds_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrewhowdencom/x40.link/api/auth/tokens/seeds"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

type roundTripper struct {
	h http.Handler
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	rt.h.ServeHTTP(w, req)
	return w.Result(), nil
}

func TestSeedDeviceAuth(t *testing.T) {
	for _, tc := range []struct {
		name string

		client *http.Client

		tok *oauth2.Token
		err error
	}{
		{
			name: "failed to generate code",
			client: &http.Client{
				Transport: &roundTripper{
					h: func() http.Handler {
						mux := chi.NewMux()

						mux.Post("/oauth/device/code", func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusInternalServerError)
						})

						return mux
					}(),
				},
			},
			tok: nil,
			err: seeds.ErrFailedToGetDeviceURL,
		},
		{
			name: "failed to exchange token",
			client: &http.Client{
				Transport: &roundTripper{
					h: func() http.Handler {
						mux := chi.NewMux()

						mux.Post("/oauth/device/code", func(w http.ResponseWriter, _ *http.Request) {
							auth := &oauth2.DeviceAuthResponse{
								DeviceCode:              "TEST-DEVICE-CODE",
								UserCode:                "TEST-USER-CODE",
								VerificationURI:         "https://x40.local/activate",
								VerificationURIComplete: "https://x40.local/activate?code=TEST-USER-CODE",
								Expiry:                  time.Now().Add(time.Hour),
								Interval:                1,
							}

							m := json.NewEncoder(w)
							if err := m.Encode(auth); err != nil {
								panic("could not write to body: " + err.Error())
							}
						})

						mux.Post("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusInternalServerError)
						})

						return mux
					}(),
				},
			},
			tok: nil,
			err: seeds.ErrFailedToExchange,
		},
		{
			name: "all ok",
			client: &http.Client{
				Transport: &roundTripper{
					h: func() http.Handler {
						mux := chi.NewMux()

						mux.Post("/oauth/device/code", func(w http.ResponseWriter, _ *http.Request) {
							auth := &oauth2.DeviceAuthResponse{
								DeviceCode:              "TEST-DEVICE-CODE",
								UserCode:                "TEST-USER-CODE",
								VerificationURI:         "https://x40.local/activate",
								VerificationURIComplete: "https://x40.local/activate?code=TEST-USER-CODE",
								Expiry:                  time.Now().Add(time.Hour),
								Interval:                1,
							}

							m := json.NewEncoder(w)
							if err := m.Encode(auth); err != nil {
								panic("could not write to body: " + err.Error())
							}
						})

						mux.Post("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Set("Content-Type", "application/json")

							tok := &oauth2.Token{
								AccessToken:  "TEST-ACCESS-TOKEN",
								RefreshToken: "TEST-REFRESH-TOKEN",
								Expiry:       time.Now().Add(time.Hour),
								TokenType:    "Bearer",
							}

							m := json.NewEncoder(w)
							if err := m.Encode(tok); err != nil {
								panic("could not write to body " + err.Error())
							}
						})

						return mux
					}(),
				},
			},
			tok: &oauth2.Token{
				AccessToken: "TEST-ACCESS-TOKEN",
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, tc.client)

			seed := seeds.DeviceAuth("https://www.example.local", &oauth2.Config{
				Endpoint: oauth2.Endpoint{
					DeviceAuthURL: "https://x40.local/oauth/device/code",
					TokenURL:      "https://x40.local/oauth/token",
				},
				ClientID: "TEST-CLIENT-ID",
			})

			tok, err := seed(ctx)

			assert.ErrorIs(t, err, tc.err)

			if tc.tok == nil {
				assert.Nil(t, tok)
			} else {
				// Here, we're just validating that we get back *a* token. The rest of it is validated by the oAuth library itself.
				assert.Equal(t, tc.tok.AccessToken, tok.AccessToken)
			}
		})
	}

}
