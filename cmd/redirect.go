/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package cmd

import (
	"github.com/andrewhowdencom/x40.link/cmd/redirect"
	"github.com/spf13/cobra"
)

// redirectCmd represents the redirect command
var redirectCmd = &cobra.Command{
	Use:   "redirect",
	Short: "Subcommands associated with the redirect server",
}

func init() {
	redirectCmd.AddCommand(redirect.Serve)
}
