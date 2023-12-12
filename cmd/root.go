/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:   "s3k.link",
	Short: "Links for Skinks",
	Long: `A short link service. Redirects users to longer links based on an
input code.

A secondary purpose of this application is to demonstrate that I (Andrew Howden)
can indeed write code. If prospective employers come looking, here's some
code!`,
	RunE: Noop,
}

func init() {
	Root.AddCommand(redirectCmd)
}
