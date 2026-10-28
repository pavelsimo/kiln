package cmd

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the kiln version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Println("kiln " + Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
