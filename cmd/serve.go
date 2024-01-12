package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/configuration"
	"github.com/andrewhowdencom/x40.link/server"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/boltdb"
	fsdb "github.com/andrewhowdencom/x40.link/storage/firestore"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/andrewhowdencom/x40.link/storage/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Sentinal errors
var (
	ErrUnsupportedStorage = errors.New("storage unsupported")
	ErrFailedStorageSetup = errors.New("failed to setup storage")
)

var (
	serveFlagSet = &pflag.FlagSet{}
)

var storageFlags = []string{
	configuration.StorageHashMap,
	configuration.StorageYamlFile,
	configuration.StorageBoltDBFile,
	configuration.StorageFirestoreProject,
}

// serveCmd starts the HTTP server that will redirect a given HTTP request to a destination.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server that handles redirects",
	RunE:  RunServe,
}

func init() {
	// Specify the address on which to listen
	serveFlagSet.StringP(configuration.ServerListenAddress, "l", "localhost:80", "The address on which to listen to incoming requests")

	// Allow providing the YAML based storage engine
	serveFlagSet.StringP(configuration.StorageYamlFile, "y", "", "Use the supplied source file as a 'yaml storage'")

	// Allow providing the in-memory based storage engine
	serveFlagSet.BoolP(configuration.StorageHashMap, "m", false, "Use in-memory (hashmap) storage")
	serveFlagSet.Lookup(configuration.StorageHashMap).NoOptDefVal = "true"

	// Allow using a file backed storage
	serveFlagSet.StringP(configuration.StorageBoltDBFile, "b", "/usr/local/share/x40/urls.db", "The place to store the URL Database")

	// External Services
	serveFlagSet.StringP(configuration.StorageFirestoreProject, "f", "", "Use the firestore database at project <input>")

	// API configuration
	serveFlagSet.StringP(configuration.ServerAPIGRPCHost, "g", "", "The host on which to listen for GRPC requests (* for all)")
	serveFlagSet.StringP(configuration.ServerAPIHTTPHost, "j", "", "The host on which to listen to HTTP+JSON requests (* for all)")

	// Protocol configuration
	serveFlagSet.BoolP(configuration.ServerH2CEnabled, "c", true, "Whether to enable HTTP/2 cleartext (with prior knowledge)")

	if err := viper.BindPFlags(serveFlagSet); err != nil {
		panic("cannot create flag: " + err.Error())
	}

	// Bind the flag set to the command, and ensure it validated.
	serveCmd.Flags().AddFlagSet(serveFlagSet)
	serveCmd.MarkFlagsOneRequired(storageFlags...)
	serveCmd.MarkFlagsMutuallyExclusive(storageFlags...)
}

// RunServe implements the run server command
func RunServe(cmd *cobra.Command, _ []string) error {
	str, err := getStorage(cmd.Flags())
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	args := []server.Option{}

	if viper.GetBool(configuration.ServerH2CEnabled) {
		args = append(args, server.WithH2C())
	}

	jHost, gHost :=
		viper.GetString(configuration.ServerAPIHTTPHost),
		viper.GetString(configuration.ServerAPIGRPCHost)

	if jHost == "*" {
		args = append(args, server.WithGRPCGateway())
	} else if jHost != "" {
		args = append(args, server.WithGRPCGateway(server.IsHost(jHost)))
	}

	if gHost == "*" {
		args = append(args, server.WithGRPC())
	} else if gHost != "" {
		args = append(args, server.WithGRPC(server.IsHost(gHost)))
	}

	args = append(args,
		server.WithStorage(str),
		server.WithListenAddress(viper.GetString(configuration.ServerListenAddress)),
	)

	srv, err := server.New(args...)
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	return srv.ListenAndServe()
}

// getStorage fetches the appropriate storage for the supplied configuration. Assumes that at least one configuration
// is passed (enforced by MarkFlagsOneRequired)
func getStorage(flags *pflag.FlagSet) (storage.Storer, error) {
	var str storage.Storer
	for _, f := range storageFlags {
		if !flags.Lookup(f).Changed {
			continue
		}

		switch f {
		case configuration.StorageHashMap:
			// It is possible, in principle, for the user to supply the flag but to be disabling the
			// option rather than just including it.
			if !viper.GetBool(configuration.StorageHashMap) {
				continue
			}

			return memory.NewHashTable(), nil
		case configuration.StorageYamlFile:
			f, err := os.Open(viper.GetString(configuration.StorageYamlFile))
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			y, err := yaml.New(memory.NewHashTable(), f)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			return y, nil
		case configuration.StorageBoltDBFile:
			db, err := boltdb.New(viper.GetString(configuration.StorageBoltDBFile))
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			return db, nil
		case configuration.StorageFirestoreProject:
			client, err := firestore.NewClient(
				context.Background(),
				viper.GetString(configuration.StorageFirestoreProject),
			)

			if err != nil {
				return nil, fmt.Errorf("%w: %s", boltdb.ErrFailedToSetupDatabase, err)
			}

			return fsdb.Firestore{
				Client: client,
			}, nil
		default:
			return nil, ErrUnsupportedStorage
		}
	}

	if str == nil {
		return nil, ErrFailedStorageSetup
	}

	return str, nil
}
