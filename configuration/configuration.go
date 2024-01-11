// Package configuration lists all of the appropriate configuration options, sets defaults and so on.
package configuration

// Server* is configuration that modifies how the server is run
const (
	ServerListenAddress  = "server.listen-address"
	ServerHTTPAPIEnabled = "server.api.http.enabled"
	ServerGRPCAPIEnabled = "server.api.grpc.enabled"

	ServerH2CEnabled = "server.protocol.h2c.enabled"
)

// Storage* is configuration related to the link storage logic.
const (
	StorageYamlFile   = "storage.yaml.file"
	StorageHashMap    = "storage.hash-map"
	StorageBoltDBFile = "store.boltdb.file"
)
