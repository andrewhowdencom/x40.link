// Package configuration lists all of the appropriate configuration options, sets defaults and so on.
package configuration

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
