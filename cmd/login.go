package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/api"
	"github.com/teal-bauer/specter/internal/config"
)

var loginCmd = &cobra.Command{
	Use:   "login [profile-name]",
	Short: "Configure specter with your Ghost credentials",
	Long: `Interactive setup to configure specter with your Ghost site.

This will:
1. Ask for your Ghost site URL
2. Open your browser to create an API integration
3. Save your credentials to ~/.config/specter/config.yaml

Examples:
  specter login              # Set up default profile
  specter login myblog       # Set up profile named "myblog"
  specter login work --default  # Set up "work" as the default profile

Then use with:
  specter posts list                # Uses default profile
  specter -p myblog posts list      # Uses "myblog" profile`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogin,
}

var (
	loginNoBrowser bool
	loginDefault   bool
)

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "Don't open browser automatically")
	loginCmd.Flags().BoolVar(&loginDefault, "default", false, "Set this profile as default")
}

func runLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Determine profile name
	profileName := "default"
	if len(args) > 0 {
		profileName = args[0]
	}

	fmt.Printf("Setting up profile: %s\n", profileName)
	fmt.Println()

	// Get Ghost URL
	fmt.Print("Enter your Ghost site URL (e.g., https://myblog.com): ")
	ghostURL, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	ghostURL = strings.TrimSpace(ghostURL)

	// Normalize URL
	if !strings.HasPrefix(ghostURL, "http://") && !strings.HasPrefix(ghostURL, "https://") {
		ghostURL = "https://" + ghostURL
	}
	ghostURL = strings.TrimSuffix(ghostURL, "/")

	// Open browser to integrations page
	integrationsURL := ghostURL + "/ghost/#/settings/integrations/new"
	fmt.Println()
	fmt.Println("To get an Admin API key, you need to create a custom integration in Ghost.")
	fmt.Println()

	if !loginNoBrowser {
		fmt.Printf("Opening: %s\n", integrationsURL)
		fmt.Println()
		if err := openBrowser(integrationsURL); err != nil {
			fmt.Printf("Could not open browser. Please visit manually:\n  %s\n", integrationsURL)
		}
	} else {
		fmt.Printf("Please visit:\n  %s\n", integrationsURL)
	}

	fmt.Println()
	fmt.Println("In Ghost Admin:")
	fmt.Println("  1. Click 'Add custom integration'")
	fmt.Println("  2. Name it 'specter' (or anything you like)")
	fmt.Println("  3. Copy the 'Admin API Key'")
	fmt.Println()

	fmt.Print("Paste your Admin API Key here: ")
	adminKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	adminKey = strings.TrimSpace(adminKey)

	if adminKey == "" {
		return fmt.Errorf("admin key cannot be empty")
	}

	// Validate the key format
	if !strings.Contains(adminKey, ":") {
		return fmt.Errorf("invalid key format: expected 'id:secret' format")
	}

	// Test the connection
	fmt.Println()
	fmt.Println("Testing connection...")

	client := api.NewClient(&config.Config{URL: ghostURL, Key: adminKey})

	data, err := client.Get("/site/", nil)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	var siteResp struct {
		Site struct {
			Title string `json:"title"`
		} `json:"site"`
	}
	if err := json.Unmarshal(data, &siteResp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Printf("Connected to: %s\n", siteResp.Site.Title)
	fmt.Println()

	// Save config
	cfg := config.Config{
		URL: ghostURL,
		Key: adminKey,
	}

	if err := config.SaveInstance(profileName, cfg, loginDefault); err != nil {
		return err
	}

	fmt.Printf("Saved profile '%s' to: %s\n", profileName, config.ConfigPath())
	fmt.Println()
	fmt.Println("You're all set! Try running:")
	if profileName == "default" {
		fmt.Println("  specter posts list")
		fmt.Println("  specter site info")
	} else {
		fmt.Printf("  specter -p %s posts list\n", profileName)
		fmt.Printf("  specter -p %s site info\n", profileName)
	}

	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
