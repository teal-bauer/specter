package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "specter",
	Short: "CLI for the Ghost Admin API",
	Long: `specter is a command-line interface for managing Ghost blogs via the Admin API.

Configure with environment variables:
  GHOST_URL        Your Ghost site URL
  GHOST_ADMIN_KEY  Admin API key (from Ghost Admin → Settings → Integrations)

Or use a config file at ~/.config/specter/config.yaml or ~/.specter.yaml:
  url: https://myblog.com
  key: "64xxxxx:xxxxxxxxxxxxxx"`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&config.FlagURL, "url", "", "Ghost site URL")
	rootCmd.PersistentFlags().StringVar(&config.FlagKey, "key", "", "Ghost Admin API key")
	rootCmd.PersistentFlags().StringVarP(&config.FlagOutput, "output", "o", "text", "Output format: text or json")
	rootCmd.PersistentFlags().StringVarP(&config.FlagProfile, "profile", "p", "", "Config profile to use")
}
