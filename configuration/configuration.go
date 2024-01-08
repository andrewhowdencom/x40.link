// package configuration lists all of the appropriate configuration options, sets defaults and so on.
package configuration

const (
	// Storage* is configuration related to the link storage logic.
	StorageYamlFile   = "storage.yaml.file"
	StorageHashMap    = "storage.hash-map"
	StorageBoltDBFile = "store.boltdb.file"

	// Server* is configuration that modifies how the server is run
	ServerListenAddress = "server.listen-address"
)
