package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cretz/owncast/src/cert"
	"github.com/cretz/owncast/src/server"
	"github.com/spf13/cobra"
)

func init() {
	serveCmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load root CA
			rootCA, err := cert.LoadFromFiles(filepath.Join("ca.crt"), filepath.Join("ca.key"))
			if err != nil {
				return fmt.Errorf("Failed loading ca.crt/ca.key, did you forget to run 'patch'? Err: %v", err)
			}
			// Start server
			srv, err := server.Listen(&server.Conf{RootCACert: rootCA})
			if err != nil {
				return fmt.Errorf("Unable to start server: %v", err)
			}
			// Use interactively
			return server.RunServerInteractively(srv, server.StdioUserInput)
		},
	}
	rootCmd.AddCommand(serveCmd)
}
