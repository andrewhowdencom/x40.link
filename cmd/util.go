package cmd

import (
	"errors"
	"fmt"

	"github.com/andrewhowdencom/sysexits"
	"github.com/spf13/cobra"
)

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

// The noop function simply indicates that this command, in and of itself, does nothing.
func Noop(c *cobra.Command, args []string) error {
	return fmt.Errorf("%w: required subcommand not supplied", sysexits.Usage)
}
