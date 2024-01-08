// Package cmd provides the top level commands for the application
package cmd

import (
	"github.com/spf13/cobra"
)

// Root is the root command for this program
var Root = &cobra.Command{
	Use:   "x40.link",
	Short: "Links for Skinks",
	Long: `A short link service. Redirects users to longer links based on an
input code.

A secondary purpose of this application is to demonstrate that I (Andrew Howden)
can indeed write code. If prospective employers come looking, here's some
code!`,
}

func init() {
	Root.AddCommand(redirectCmd)
}
