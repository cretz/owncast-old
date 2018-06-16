package cmd

import (
	"fmt"
	"os"

	"github.com/cretz/owncast/owncast/log"
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
			log.Debugf = log.SimpleLogf
		} else if quiet {
			log.Infof = log.NoopLogf
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&certDir,
		"cert-dir", "d", ".", "Sets the dir to load/create ca.crt and ca.key")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug logs")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Hide info logs")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
