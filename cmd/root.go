/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package cmd

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/sysexits"
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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("This application does nothing just yet.")
	},
}

// Execute runs a (root) command, and returns an enriched "error" which describes the exit status of the application.
// Essentially, a utility that allows this to be validated with tests.
func Execute(c *cobra.Command) sysexits.Sysexit {
	err := c.Execute()

	// Success
	if err == nil {
		return sysexits.OK
	}

	// Check if the program hass passed an error back, enriched with context that allows deciding how to exit.
	var exit sysexits.Sysexit
	if errors.As(err, &exit) {
		return exit
	}

	// The default (software error)
	return sysexits.Software
}

// init func to declare configuration
func init() {
}
