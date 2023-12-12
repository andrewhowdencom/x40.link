/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package cmd

import (
	"github.com/andrewhowdencom/s3k.link/cmd/redirect"
	"github.com/spf13/cobra"
)

// redirectCmd represents the redirect command
var redirectCmd = &cobra.Command{
	Use:   "redirect",
	Short: "Subcommands associated with the redirect server",
	RunE:  Noop,
}

func init() {
	redirectCmd.AddCommand(redirect.Serve)
}
