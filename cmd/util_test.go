package cmd_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/andrewhowdencom/sysexits"
	"github.com/andrewhowdencom/x40.link/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Validate that the applications various internal failures will be correctly reflected via the appropriate exit
// code.
func TestExecute(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
		exit sysexits.Sysexit
	}{
		{
			name: "Everything OK",
			cmd: &cobra.Command{
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			exit: sysexits.OK,
		},
		{
			name: "Passed a sysexit back",
			cmd: &cobra.Command{
				RunE: func(cmd *cobra.Command, args []string) error {
					return fmt.Errorf("its broke: %w", sysexits.Unavailable)
				},
			},
			exit: sysexits.Unavailable,
		},
		{
			name: "Passed no sysexit, but had an error",
			cmd: &cobra.Command{
				RunE: func(cmd *cobra.Command, args []string) error {
					return errors.New("I am very mysterious")
				},
			},
			exit: sysexits.Software,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()

			exit := cmd.Execute(tc.cmd)

			assert.ErrorIs(t, exit, tc.exit)
		})
	}
}
