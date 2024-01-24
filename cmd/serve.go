package cmd

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Sentinal errors
var (
	ErrUnsupportedStorage = errors.New("storage unsupported")
)

var (
	serveFlagSet = &pflag.FlagSet{}
)

var storageFlags = []string{
	cfg.StorageHashMap.Path,
	cfg.StorageYamlFile.Path,
	cfg.StorageBoltDBFile.Path,
	cfg.StorageFirestoreProject.Path,
}

// serveCmd starts the HTTP server that will redirect a given HTTP request to a destination.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server that handles redirects",
	RunE:  RunServe,
}

func init() {
	for _, f := range []interface {
		AddFlagTo(fs *pflag.FlagSet)
	}{
		// Storage Flags
		cfg.StorageYamlFile,
		cfg.StorageHashMap,
		cfg.StorageBoltDBFile,
		cfg.StorageFirestoreProject,

		// Authentication
		cfg.AuthX40,

		cfg.AuthJWKSURL,
		cfg.AuthClaimIssuer,
		cfg.AuthClaimAudience,
		cfg.AuthClaimIssuedAt,
		cfg.AuthClaimExpiration,

		// Server
		cfg.ServerListenAddress,

		cfg.ServerAPIGRPCHost,
		cfg.ServerH2CEnabled,
	} {
		f.AddFlagTo(serveFlagSet)
	}

	// Bind the flag set to the command, and ensure it validated.
	serveCmd.Flags().AddFlagSet(serveFlagSet)
	serveCmd.MarkFlagsOneRequired(storageFlags...)
	serveCmd.MarkFlagsMutuallyExclusive(storageFlags...)

}

// RunServe implements the run server command
func RunServe(_ *cobra.Command, _ []string) error {
	srv, err := server.WireServer()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	return srv.ListenAndServe()
}
