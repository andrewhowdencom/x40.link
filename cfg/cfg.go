// Package cfg lists all of the appropriate configuration options, sets defaults and so on.
package cfg

import (
	"errors"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// V  is Value
type V struct {
	// Path is the json notation path that this configuration is available at
	Path string

	// Usage is the (max 72 character) Usage for a given configuration item
	Usage string

	// Short is an optional, Short string (1 char) that can be used to identify this configuration
	// in a limited set (such as flags). If zero length, is skipped.
	Short string

	// Default is the default value for this configuration item. Can be any.
	Default interface{}

	mu *sync.Mutex
}

// Bool is a configuration entry that is a boolean value
type Bool struct {
	V
}

// Value returns the value of the configuration
func (b *Bool) Value() bool {
	if !viper.IsSet(b.Path) {
		return b.Default.(bool)
	}

	return viper.GetBool(b.Path)
}

// String is the string implementation
type String struct {
	V
}

// Value returns the value of the configuration
func (s String) Value() string {
	if !viper.IsSet(s.Path) {
		return s.Default.(string)
	}

	return viper.GetString(s.Path)
}

var (
	// ErrMissingOptions can be used by packages to indicate that whatever option they were looking for isn't
	// present in the configuration, or in the expected format.
	//
	// Used primarily as part of dependency injection to skip optional dependencies.
	ErrMissingOptions = errors.New("required options missing")
)

// The actual configuration values.
var (
	APIEndpoint = &String{V: V{Path: "api.endpoint", Default: "https://api.x40.link", Usage: "The endpoint to talk to for links", mu: &sync.Mutex{}}}

	// AuthX40 just means "authenticate this against the public X40 endpoints"
	AuthX40 = &Bool{V: V{Path: "auth.x40", Default: false, Usage: "Whether to configure the application to authenticate against the public x40 links", mu: &sync.Mutex{}}}

	// The JWKSURL to point the authentication to
	AuthJWKSURL = &String{V: V{Path: "auth.jwks.url", Default: "", Usage: "The endpoint that fetches key material to validate JWT tokens", mu: &sync.Mutex{}}}

	// Specific enforcement for the JWTs
	AuthClaimIssuer     = &String{V: V{Path: "auth.jwt.issuer", Default: "", Usage: "The issuer of JWT tokens, validated in the authentication", mu: &sync.Mutex{}}}
	AuthClaimAudience   = &String{V: V{Path: "auth.jwt.audience", Default: "", Usage: "The audience (or app) the JWT is issued for, validated in the authentication", mu: &sync.Mutex{}}}
	AuthClaimIssuedAt   = &Bool{V: V{Path: "auth.jwt.issued-at", Default: false, Usage: "The time at which a JWT was issued, validated in the authentication", mu: &sync.Mutex{}}}
	AuthClaimExpiration = &Bool{V: V{Path: "auth.jwt.expiration", Default: true, Usage: "Whether to force expiration on tokens", mu: &sync.Mutex{}}}

	// OAuth2 configuration. Configured to point to production systems by default.
	OAuth2AuthorizationURL = &String{V: V{Path: "oauth2.authorization.url", Default: "https://x40.eu.auth0.com/authorize", Usage: "The URL for the authorization flow", mu: &sync.Mutex{}}}

	// x40.auth0/x40-cli
	//
	// Safe to embed. See:
	// https://www.oauth.com/oauth2-servers/client-registration/client-id-secret/
	OAuth2ClientID                    = &String{V: V{Path: "oauth2.client.id", Default: "FH72Qo7CrVKE9hr71cHYKbLimKAobMot", Usage: "The ClientID to present during the authorization flow", mu: &sync.Mutex{}}}
	OAuth2DeviceAuthorizationEndpoint = &String{V: V{Path: "oauth2.device-authorization.url", Default: "https://x40.eu.auth0.com/oauth/device/code", Usage: "The URL for the device flow", mu: &sync.Mutex{}}}
	OAuth2TokenURL                    = &String{V: V{Path: "oauth2.token.url", Default: "https://x40.eu.auth0.com/oauth/token", Usage: "The URL that can be used to exchange auth for tokens", mu: &sync.Mutex{}}}

	ServerListenAddress = &String{V: V{Path: "server.listen-address", Default: "localhost:80", Usage: "The address on which to listen to incoming requests", mu: &sync.Mutex{}}}
	ServerAPIGRPCHost   = &String{V: V{Path: "server.api.grpc.host", Default: "", Usage: "The host on which to listen to GRPC requests (* means all)", mu: &sync.Mutex{}}}
	ServerH2CEnabled    = &Bool{V: V{Path: "server.protocol.h2c.enabled", Default: true, Usage: "Whether to enable the HTTP/2 Cleartext (with prior knowledge)", mu: &sync.Mutex{}}}

	// Storage* is configuration related to the link storage logic.
	StorageYamlFile         = &V{Path: "storage.yaml.file", Default: "", Usage: "The source file to read URLs from", mu: &sync.Mutex{}}
	StorageHashMap          = &V{Path: "storage.hash-map", Default: false, Usage: "Whether to use an in-memory hash map as URL storage", mu: &sync.Mutex{}}
	StorageBoltDBFile       = &V{Path: "storage.boltdb.file", Default: "", Usage: "The source file to use with boldDB backed URL storage", mu: &sync.Mutex{}}
	StorageFirestoreProject = &V{Path: "storage.firestore.project", Default: "", Usage: "The Google Cloud project to use the default firebase storage for", mu: &sync.Mutex{}}

	Timeout = &String{V: V{Path: "timeout", Default: "1m", Usage: "The fallback timeout across the application", mu: &sync.Mutex{}}}
)

// AddFlagTo accepts a flag set, and adds the flag to it. It also binds that flag to the Viper configuration.
//
// Not concurrency safe (underlying library races in pflag)
func (v *V) AddFlagTo(fs *pflag.FlagSet) {
	v.mu.Lock()
	defer v.mu.Unlock()

	switch v.Default.(type) {
	case string:
		fs.StringP(v.Path, v.Short, v.Default.(string), v.Usage)
	case bool:
		fs.BoolP(v.Path, v.Short, v.Default.(bool), v.Usage)
	default:
		panic("unsupported conversion to flag: " + v.Path)
	}

	if err := viper.BindPFlag(v.Path, fs.Lookup(v.Path)); err != nil {
		panic("failed to bind p flag " + err.Error())
	}
}
