/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package main

import (
	"os"

	"github.com/andrewhowdencom/x40.link/cmd"
)

func main() {
	// Cobra will print the exit.String() as part of its Execute method. Here, we only need to check
	// what the exit code should be.
	exit := cmd.Execute(cmd.Root)
	os.Exit(exit.Code)
}
