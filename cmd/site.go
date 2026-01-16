package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/api"
	"github.com/teal-bauer/specter/internal/config"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Site information",
}

var siteInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get site information",
	RunE:  runSiteInfo,
}

func init() {
	rootCmd.AddCommand(siteCmd)
	siteCmd.AddCommand(siteInfoCmd)
}

type Site struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Logo        string `json:"logo,omitempty"`
	Icon        string `json:"icon,omitempty"`
	URL         string `json:"url"`
	Version     string `json:"version"`
}

type siteResponse struct {
	Site Site `json:"site"`
}

func runSiteInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	data, err := client.Get("/site/", nil)
	if err != nil {
		return err
	}

	var resp siteResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Site)
	}

	fmt.Printf("Title:       %s\n", resp.Site.Title)
	fmt.Printf("Description: %s\n", resp.Site.Description)
	fmt.Printf("URL:         %s\n", resp.Site.URL)
	fmt.Printf("Version:     %s\n", resp.Site.Version)
	if resp.Site.Logo != "" {
		fmt.Printf("Logo:        %s\n", resp.Site.Logo)
	}
	if resp.Site.Icon != "" {
		fmt.Printf("Icon:        %s\n", resp.Site.Icon)
	}
	return nil
}
