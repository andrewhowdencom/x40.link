/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// init func to declare configuration
func init() {
}
