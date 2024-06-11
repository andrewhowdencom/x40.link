// package main is the main package of the CLI client
package main

import (
	"context"
	"fmt"
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
	"google.golang.org/grpc/metadata"
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
	Args:  cobra.ExactArgs(1),
	RunE:  DoURL,
}

// DoURL is the root command for the client, and generates URLs
func DoURL(_ *cobra.Command, args []string) error {
	ts, err := auth.TokenSource()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	client, err := api.NewGRPCClient(viper.GetString(cfg.APIEndpoint.Path))
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.NoHost, err)
	}

	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("%w: %s", sysexits.Software, err)
	}

	md := metadata.New(map[string]string{
		"Authorization": "Bearer " + tok.AccessToken,
	})

	ctx, cxl := context.WithTimeout(context.Background(), time.Second*10)
	defer cxl()

	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.New(ctx, &dev.NewRequest{
		SendTo: args[0],
	})

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
