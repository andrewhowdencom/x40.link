// Package redirect provides the commands associated with redirecting users
package redirect

import (
	"errors"
	"fmt"
	"os"

	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/configuration"
	"github.com/andrewhowdencom/x40.link/server"
	"github.com/andrewhowdencom/x40.link/storage"
	"github.com/andrewhowdencom/x40.link/storage/boltdb"
	"github.com/andrewhowdencom/x40.link/storage/memory"
	"github.com/andrewhowdencom/x40.link/storage/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	flagStrHashMap = "with-hash-map"
	flagStrYAML    = "with-yaml"
	flagStrBoltDB  = "with-boltdb"

	flagStrListenAddress = "listen-address"
)

// Sentinal errors
var (
	ErrUnsupportedStorage = errors.New("storage unsupported")
	ErrFailedStorageSetup = errors.New("failed to setup storage")
)

var (
	serveFlagSet = &pflag.FlagSet{}
)

var storageFlags = []string{flagStrHashMap, flagStrYAML, flagStrBoltDB}

// Serve starts the HTTP server that will redirect a given HTTP request to a destination.
var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Start the server that handles redirects",
	RunE:  RunServe,
}

func init() {
	// Specify the address on which to listen
	serveFlagSet.StringP(flagStrListenAddress, "l", "localhost:80", "The address on which to listen to incoming requests")

	// Allow providing the YAML based storage engine
	serveFlagSet.StringP(flagStrYAML, "y", "", "Use the supplied source file as a 'yaml storage'")

	// Allow providing the in-memory based storage engine
	serveFlagSet.BoolP(flagStrHashMap, "m", false, "Use in-memory (hashmap) storage")
	serveFlagSet.Lookup(flagStrHashMap).NoOptDefVal = "true"

	// Allow using a file backed storage
	serveFlagSet.StringP(flagStrBoltDB, "b", "/usr/local/share/x40/urls.db", "The place to store the URL Database")

	// Bind the flags to the configuration
	for c, f := range map[string]*pflag.Flag{
		configuration.StorageYamlFile:   serveFlagSet.Lookup(flagStrYAML),
		configuration.StorageHashMap:    serveFlagSet.Lookup(flagStrHashMap),
		configuration.StorageBoltDBFile: serveFlagSet.Lookup(flagStrBoltDB),

		configuration.ServerListenAddress: serveFlagSet.Lookup(flagStrListenAddress),
	} {
		if err := viper.BindPFlag(c, f); err != nil {
			panic("cannot create flag: " + err.Error())
		}
	}

	// Bind the flag set to the command, and ensure it validated.
	Serve.Flags().AddFlagSet(serveFlagSet)
	Serve.MarkFlagsOneRequired(storageFlags...)
	Serve.MarkFlagsMutuallyExclusive(storageFlags...)
}

// RunServe implements the run server command
func RunServe(cmd *cobra.Command, _ []string) error {
	str, err := getStorage(cmd.Flags())
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	srv, err := server.New(
		server.WithStorage(str),
		server.WithListenAddress(viper.GetString(configuration.ServerListenAddress)),
	)
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	return srv.Start()
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
		case flagStrHashMap:
			// It is possible, in principle, for the user to supply the flag but to be disabling the
			// option rather than just including it.
			if !viper.GetBool(configuration.StorageHashMap) {
				continue
			}

			return memory.NewHashTable(), nil
		case flagStrYAML:
			f, err := os.Open(viper.GetString(configuration.StorageYamlFile))
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			y, err := yaml.New(memory.NewHashTable(), f)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			return y, nil
		case flagStrBoltDB:
			db, err := boltdb.New(viper.GetString(configuration.StorageBoltDBFile))
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedStorageSetup, err)
			}

			return db, nil
		default:
			return nil, ErrUnsupportedStorage
		}
	}

	if str == nil {
		return nil, ErrFailedStorageSetup
	}

	return str, nil
}
