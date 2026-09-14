package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/otel"
	"github.com/andrewhowdencom/x40.link/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// initTelemetry is the seam used by RunServe to construct the OTel
// TracerProvider / MeterProvider. It is exposed only to tests in this
// package; the production caller is RunServe.
func initTelemetry(ctx context.Context) (func(context.Context) error, error) {
	return otel.Init(ctx)
}

// shutdownTimeout bounds how long RunServe waits for in-flight spans
// and metrics to flush on shutdown. The OTel SDK's batch span processor
// and the periodic reader both honour context cancellation, so this
// cap is sufficient to prevent a hung exporter from blocking process
// exit indefinitely.
const shutdownTimeout = 5 * time.Second

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

		// OpenTelemetry
		cfg.OTELEnabled,
		cfg.OTELExporterEndpoint,
		cfg.OTELExporterInsecure,
		cfg.OTELProbeEndpoint,
		cfg.OTELServiceName,
		cfg.OTELResourceAttributes,
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialise OpenTelemetry as the first thing we do. If Init fails,
	// surface the failure as a sysexits-Software error so the cobra
	// Execute path can extract it cleanly.
	otelShutdown, err := initTelemetry(ctx)
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	// Defer the OTel shutdown with a bounded timeout so a hung exporter
	// cannot block process exit. Errors from the shutdown are logged via
	// the standard library log package and not propagated, because at
	// this point the only caller of RunServe has already received the
	// server's error (or nil) and there is no higher-level caller that
	// could meaningfully act on a shutdown error.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			log.Printf("otel: shutdown: %v", err)
		}
	}()

	srv, err := server.WireServer()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	return srv.ListenAndServe()
}
