/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package redirect

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/andrewhowdencom/s3k.link/configuration"
	"github.com/andrewhowdencom/s3k.link/storage"
	"github.com/andrewhowdencom/s3k.link/storage/boltdb"
	"github.com/andrewhowdencom/s3k.link/storage/memory"
	"github.com/andrewhowdencom/s3k.link/storage/yaml"
	"github.com/andrewhowdencom/sysexits"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	flagStrHashMap = "with-hash-map"
	flagStrYAML    = "with-yaml"
	flagStrBoltDB  = "with-boltdb"
)

// Sentinal errors
var (
	ErrUnsupportedStorage = errors.New("storage unsupported")
	ErrFailedStorageSetup = errors.New("failed to setup storage")
)

var (
	serveFlagSet = &pflag.FlagSet{}
)

var strFlags = []string{flagStrHashMap, flagStrYAML}

// Serve starts the HTTP server that will redirect a given HTTP request to a destination.
var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Start the server that handles redirects",
	RunE:  RunServe,
}

func init() {

	// Allow providing the YAML based storage engine
	serveFlagSet.StringP(flagStrYAML, "y", "", "Use the supplied source file as a 'yaml storage'")
	viper.BindPFlag(configuration.StorageYamlFile, serveFlagSet.Lookup(flagStrYAML))

	// Allow providing the in-memory based storage engine
	serveFlagSet.BoolP(flagStrHashMap, "m", false, "Use in-memory (hashmap) storage")
	serveFlagSet.Lookup(flagStrHashMap).NoOptDefVal = "true"
	viper.BindPFlag(configuration.StorageHashMap, serveFlagSet.Lookup(flagStrHashMap))

	serveFlagSet.StringP(flagStrBoltDB, "b", "/usr/local/share/s3k/urls.db", "The place to store the URL Database")
	viper.BindPFlag(configuration.StoreBoltDBFile, serveFlagSet.Lookup(flagStrBoltDB))

	// Bind the flag set to the command, and ensure it validated.
	Serve.Flags().AddFlagSet(serveFlagSet)
	Serve.MarkFlagsOneRequired(strFlags...)
	Serve.MarkFlagsMutuallyExclusive(strFlags...)
}

func RunServe(cmd *cobra.Command, args []string) error {
	str, err := getStorage(cmd.Flags())
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	// Stub implementation to validate runtime constraints.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Figure out a saner way to do this
		url := &url.URL{
			// There's no support for anything else at this time
			Scheme: "http",
			Host:   r.Host,
			Path:   r.URL.Path,
		}
		ret, err := str.Get(url)

		// Iterate though the potential failure modes.
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Location", ret.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	return http.ListenAndServe("localhost:80", http.DefaultServeMux)
}

// getStorage fetches the appropriate storage for the supplied configuration. Assumes that at least one configuration
// is passed (enforced by MarkFlagsOneRequired)
func getStorage(flags *pflag.FlagSet) (storage.Storer, error) {
	var str storage.Storer
	for _, f := range strFlags {
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
			db, err := boltdb.New(viper.GetString(configuration.StoreBoltDBFile))
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
