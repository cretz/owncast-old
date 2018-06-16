package cmd

import (
	"fmt"
	"os"

	"github.com/cretz/owncast/src/util"
	"github.com/spf13/cobra"
)

var certDir string
var verbose bool
var quiet bool

var rootCmd = &cobra.Command{
	Use: "owncast",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if verbose && quiet {
			return fmt.Errorf("Cannot set verbose and quiet")
		} else if verbose {
			util.LogDebug = util.SimpleLogger
		} else if quiet {
			util.LogInfo = util.NoopLogger
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&certDir, "cert-dir", ".", "Sets the dir to load/create ca.crt and ca.key")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show debug logs")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Hide info logs")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
