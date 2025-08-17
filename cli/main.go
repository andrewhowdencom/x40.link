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
)

var (
	urlFlagSet = func() *pflag.FlagSet {
		fs := &pflag.FlagSet{}

		for _, f := range []interface {
			AddFlagTo(*pflag.FlagSet)
		}{
			cfg.APIEndpoint,

			cfg.OAuth2ClientID,
			cfg.OAuth2AuthorizationURL,
			cfg.OAuth2DeviceAuthorizationEndpoint,
			cfg.OAuth2TokenURL,
		} {
			f.AddFlagTo(fs)
		}

		return fs
	}()
)

// Root represents the url command
var Root = &cobra.Command{
	Use:   "@",
	Short: "The client tool for generating URLs",
	Long: `The client tool for generating URLs.

Generate a random URL on x40.link:

    @ https://my.destination.url/path

Generate a URL on a specific domain, registered on x40.link:

    @ https://source.domain/path https://my.destination.url/path

	`,
	Args: cobra.MinimumNArgs(1),
	RunE: DoURL,
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

func init() {
	Root.Flags().AddFlagSet(urlFlagSet)
}

func main() {
	// Cobra will print the exit.String() as part of its Execute method. Here, we only need to check
	// what the exit code should be.
	exit := cmd.Execute(Root)
	os.Exit(exit.Code)
}
