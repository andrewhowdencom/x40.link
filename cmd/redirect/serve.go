/*
Copyright © 2023 Andrew Howden <hello@andrewhowden.com>
*/
package redirect

import (
	"net/http"

	"github.com/spf13/cobra"
)

// Serve starts the HTTP server that will redirect a given HTTP request to a destination.
var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Start the server that handles redirects",
	RunE:  RunServe,
}

func RunServe(cmd *cobra.Command, args []string) error {
	// Stub implementation to validate runtime constraints.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	return http.ListenAndServe("localhost:80", http.DefaultServeMux)
}

func init() {
}
