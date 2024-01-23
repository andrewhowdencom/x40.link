// Package configuration lists all of the appropriate configuration options, sets defaults and so on.
//
// TODO: Restructure this as a series of structs, with names, defaults and so on. Smth like:
//
//	struct {
//		   path string
//		   description string
//		   default interface{}
//	}
package configuration

// Auth* are constants related to authenticating the service
const (
	// AuthX40 just means "authenticate this against the public X40 endpoints"
	AuthX40 = "auth.x40"

	// The JWKSURL to point the authentication to
	AuthJWKSURL = "auth.jwks.url"

	// Specific enforcement for the JWTs
	AuthClaimIssuer     = "auth.jwt.issuer"
	AuthClaimAudience   = "auth.jwt.audience"
	AuthClaimIssuedAt   = "auth.jwt.issued-at"
	AuthClaimExpiration = "auth.jwt.expiration"
)

// OAuth2* is configuration for either requesting or validating OAuth2 keys.
const (
	OAuth2ClientID     = "oauth2.client.id"
	OAuth2ClientSecret = "oauth2.client.secret"
)

// OIDC* is configuration related to OpenID (an extension, essentially, of OAuth)
const (
	OIDCProviderEndpoint = "oidc.provider.endpoint"
)

// Server* is configuration that modifies how the server is run
const (
	ServerListenAddress = "server.listen-address"
	ServerAPIGRPCHost   = "server.api.grpc.host"

	ServerH2CEnabled = "server.protocol.h2c.enabled"
)

// Storage* is configuration related to the link storage logic.
const (
	StorageYamlFile         = "storage.yaml.file"
	StorageHashMap          = "storage.hash-map"
	StorageBoltDBFile       = "storage.boltdb.file"
	StorageFirestoreProject = "storage.firestore.project"
)
