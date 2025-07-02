package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	envConfigPath     = "JELLYPORTER_CONFIG_PATH"
	defaultConfigPath = "/etc/jellyporter.yaml"
)

var (
	configPath     string
	flagConfigPath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jellyporter",
	Short: "Sync user playback data across multiple Jellyfin servers",
	Long: `jellyporter is a fast, event-driven tool for synchronizing user playback data
(UserData) — such as watched status, resume position, and playback timestamps — 
across multiple Jellyfin instances.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configPath = flagConfigPath
		if configPath == "" {
			configPath = os.Getenv(envConfigPath)
		}
		if configPath == "" {
			configPath = defaultConfigPath
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagConfigPath, "config", "c", "", "Path to YAML config file")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
