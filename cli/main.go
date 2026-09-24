// package main is the main package of the CLI client
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/api"
	"github.com/andrewhowdencom/x40.link/api/gen/dev"
	"github.com/andrewhowdencom/x40.link/cfg"
	"github.com/andrewhowdencom/x40.link/cli/auth"
	"github.com/andrewhowdencom/x40.link/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Flag sets.
//
// The CLI exposes two kinds of commands: those that need a backend gRPC client
// ("api") and those that additionally need OAuth configuration ("auth"). The
// root command (which creates a short link) needs both; "resolve" needs only the
// API endpoint, while "login" needs only OAuth configuration.
var (
	apiFlagSet = func() *pflag.FlagSet {
		fs := &pflag.FlagSet{}

		for _, f := range []interface {
			AddFlagTo(*pflag.FlagSet)
		}{
			cfg.APIEndpoint,
		} {
			f.AddFlagTo(fs)
		}

		return fs
	}()

	authFlagSet = func() *pflag.FlagSet {
		fs := &pflag.FlagSet{}

		for _, f := range []interface {
			AddFlagTo(*pflag.FlagSet)
		}{
			cfg.OAuth2ClientID,
			cfg.OAuth2AuthorizationURL,
			cfg.OAuth2DeviceAuthorizationEndpoint,
			cfg.OAuth2TokenURL,
		} {
			f.AddFlagTo(fs)
		}

		return fs
	}()

	// urlFlagSet composes both sets for the root command. Subcommands attach
	// only the configuration they need.
	urlFlagSet = func() *pflag.FlagSet {
		fs := &pflag.FlagSet{}

		fs.AddFlagSet(apiFlagSet)
		fs.AddFlagSet(authFlagSet)

		return fs
	}()
)

// resolveTimeout is the timeout used for the gRPC Get call. It is intentionally
// the same as the root command's timeout (10s); the operation is read-only and
// does not need a longer window.
const resolveTimeout = 10 * time.Second

// Root represents the url command
var Root = &cobra.Command{
	Use:   "@",
	Short: "The client tool for generating URLs",
	Long: `The client tool for generating URLs.

Generate a random URL on x40.link:

    @ https://my.destination.url/path

Generate a URL on a specific domain, registered on x40.link:

    @ https://source.domain/path https://my.destination.url/path

Or, look up the destination of an existing short link:

    @ resolve https://source.domain/path

Replace the cached login with a different account:

    @ login

	`,
	Args: cobra.MinimumNArgs(1),
	RunE: DoURL,
}

// resolveCmd is the "resolve" subcommand. It looks up the destination of a short
// link and prints it to stdout. It is intentionally unauthenticated, since the
// gRPC Get RPC it calls is publicly callable.
var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Look up the destination of a short link",
	Long: `Look up the destination of a short link.

Given a short URL, print the URL it redirects to. The command does not
require authentication; the destination of a short link is functionally
public information, since the HTTP redirect already discloses it to
anonymous users.

Example:

    @ resolve https://x40.link/abc
`,
	Args: cobra.ExactArgs(1),
	RunE: DoResolve,
}

// loginCmd is the "login" subcommand. It always runs a fresh OAuth device
// authorization flow and replaces cached credentials only after success.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with a different account",
	Long: `Log in with a different account.

Start a fresh OAuth device authorization flow. Existing cached credentials
remain available if authentication fails or is cancelled.

Example:

    @ login
`,
	Args: cobra.NoArgs,
	RunE: DoLogin,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List links visible to your account",
	Long: `List short links visible to your account.

Filter by source domain with --domain. Firestore returns links owned by
your account; storage backends without ownership data return all links.

Examples:

    @ list
    @ list --domain x40.link
`,
	Args: cobra.NoArgs,
	RunE: DoList,
}

// DoURL is the root command for the client, and generates URLs
func DoURL(_ *cobra.Command, args []string) error {

	req := &dev.NewRequest{}

	// Stub the scheme there. Only HTTPS is supported.
	for idx := range args {
		if !strings.Contains(args[idx], "://") {
			args[idx] = "https://" + args[idx]
		}
	}

	switch len(args) {
	case 1:
		req.SendTo = args[0]
	case 2:
		req.SendTo = args[1]
		u, err := url.Parse(args[0])
		if err != nil {
			return err
		}

		req.On = &dev.RedirectOn{
			Host: u.Host,
			Path: u.Path,
		}
	}

	ts, err := auth.TokenSource()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	client, err := api.NewGRPCClient(
		viper.GetString(cfg.APIEndpoint.Path),
		grpc.WithPerRPCCredentials(auth.NewPerRPCCredentials(ts)),
	)
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.NoHost, err)
	}

	ctx, cxl := context.WithTimeout(context.Background(), time.Second*10)
	defer cxl()

	resp, err := client.New(ctx, req)

	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Protocol, err)
	}

	url, _ := strings.CutPrefix(resp.Url, "//")
	fmt.Println(url)

	return nil
}

// DoLogin is the cobra command handler for the "login" subcommand.
func DoLogin(cmd *cobra.Command, _ []string) error {
	if err := doLogin(cmd.Context(), auth.Login); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Login successful."); err != nil {
		return fmt.Errorf("%w: write login confirmation: %w", sysexits.Software, err)
	}

	return nil
}

func doLogin(ctx context.Context, login func(context.Context) error) error {
	if err := login(ctx); err != nil {
		return fmt.Errorf("%w: %w", sysexits.Software, err)
	}

	return nil
}

// DoResolve is the cobra command handler for the "resolve" subcommand. It builds
// a gRPC client (without per-RPC credentials, since the Get RPC is public) and
// delegates the actual call to doResolveWithClient for testability.
func DoResolve(_ *cobra.Command, args []string) error {
	client, err := api.NewGRPCClient(viper.GetString(cfg.APIEndpoint.Path))
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.NoHost, err)
	}

	ctx, cxl := context.WithTimeout(context.Background(), resolveTimeout)
	defer cxl()

	destination, err := doResolveWithClient(ctx, client, args[0])
	if err != nil {
		return err
	}

	fmt.Println(destination)
	return nil
}

// DoList fetches links through the authenticated API and prints one mapping per line.
func DoList(cmd *cobra.Command, _ []string) error {
	domain, err := cmd.Flags().GetString("domain")
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}
	ts, err := auth.TokenSource()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}
	client, err := api.NewGRPCClient(
		viper.GetString(cfg.APIEndpoint.Path),
		grpc.WithPerRPCCredentials(auth.NewPerRPCCredentials(ts)),
	)
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.NoHost, err)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	links, err := doListWithClient(ctx, client, domain)
	if err != nil {
		return err
	}
	for _, link := range links {
		source, _ := strings.CutPrefix(link.From, "//")
		destination, _ := strings.CutPrefix(link.To, "//")
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", source, destination); err != nil {
			return fmt.Errorf("%w: %s", sysexits.Software, err)
		}
	}
	return nil
}

func doListWithClient(ctx context.Context, client api.Client, domain string) ([]*dev.Link, error) {
	response, err := client.List(ctx, &dev.ListRequest{Domain: domain})
	if err != nil {
		return nil, classifyResolveError(err)
	}
	return response.Links, nil
}

// doResolveWithClient is the testable core of the resolve flow. It takes a
// ready-to-use gRPC client, an input URL string, and returns the destination
// URL (with any leading "//" stripped, matching the DoURL convention) or a
// sysexits-wrapped error appropriate to the failure mode.
//
// The "https://" scheme is prepended to inputs that have no scheme, mirroring
// the existing DoURL behavior.
func doResolveWithClient(ctx context.Context, client api.Client, input string) (string, error) {
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}

	resp, err := client.Get(ctx, &dev.GetRequest{Url: input})
	if err != nil {
		return "", classifyResolveError(err)
	}

	// Strip a leading "//" from the response to match the DoURL convention.
	destination, _ := strings.CutPrefix(resp.Url, "//")

	return destination, nil
}

// classifyResolveError maps a gRPC error from the Get call to a sysexits code.
// NotFound and InvalidArgument both indicate "the input data is wrong", which
// maps to DataErr. A bare (non-gRPC) error suggests a transport failure and
// maps to NoHost. Other gRPC errors are treated as protocol failures.
func classifyResolveError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status error — assume transport failure.
		return fmt.Errorf("%w: %s", sysexits.NoHost, err)
	}

	switch st.Code() {
	case codes.NotFound, codes.InvalidArgument:
		return fmt.Errorf("%w: %s", sysexits.DataErr, st.Message())
	default:
		return fmt.Errorf("%w: %s", sysexits.Protocol, st.Message())
	}
}

func init() {
	Root.Flags().AddFlagSet(urlFlagSet)
	Root.AddCommand(loginCmd, resolveCmd, listCmd)
	loginCmd.Flags().AddFlagSet(authFlagSet)
	resolveCmd.Flags().AddFlagSet(apiFlagSet)
	listCmd.Flags().AddFlagSet(urlFlagSet)
	listCmd.Flags().String("domain", "", "restrict results to this source domain")
}

func main() {
	// Cobra will print the exit.String() as part of its Execute method. Here, we only need to check
	// what the exit code should be.
	exit := cmd.Execute(Root)
	os.Exit(exit.Code)
}
