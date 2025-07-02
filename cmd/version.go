package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s go%s\n", BuildVersion, CommitHash, GoVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
