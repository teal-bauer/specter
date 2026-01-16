package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/api"
	"github.com/teal-bauer/specter/internal/config"
)

var tiersCmd = &cobra.Command{
	Use:   "tiers",
	Short: "Manage tiers (subscription products)",
}

var tiersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tiers",
	RunE:  runTiersList,
}

var tiersGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a tier by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runTiersGet,
}

var tiersCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a tier",
	Args:  cobra.ExactArgs(1),
	RunE:  runTiersCreate,
}

var tiersUpdateCmd = &cobra.Command{
	Use:   "update <id-or-slug>",
	Short: "Update a tier",
	Args:  cobra.ExactArgs(1),
	RunE:  runTiersUpdate,
}

var (
	tierSlug          string
	tierDescription   string
	tierMonthlyPrice  int
	tierYearlyPrice   int
	tierCurrency      string
	tierActive        string
	tierWelcomePageURL string
	tierVisibility    string
	tierTrialDays     int
)

func init() {
	rootCmd.AddCommand(tiersCmd)
	tiersCmd.AddCommand(tiersListCmd)
	tiersCmd.AddCommand(tiersGetCmd)
	tiersCmd.AddCommand(tiersCreateCmd)
	tiersCmd.AddCommand(tiersUpdateCmd)

	tiersCreateCmd.Flags().StringVar(&tierSlug, "slug", "", "Tier slug")
	tiersCreateCmd.Flags().StringVar(&tierDescription, "description", "", "Tier description")
	tiersCreateCmd.Flags().IntVar(&tierMonthlyPrice, "monthly-price", 0, "Monthly price in cents")
	tiersCreateCmd.Flags().IntVar(&tierYearlyPrice, "yearly-price", 0, "Yearly price in cents")
	tiersCreateCmd.Flags().StringVar(&tierCurrency, "currency", "usd", "Currency code")
	tiersCreateCmd.Flags().StringVar(&tierVisibility, "visibility", "public", "Visibility: public or none")
	tiersCreateCmd.Flags().IntVar(&tierTrialDays, "trial-days", 0, "Trial period in days")

	tiersUpdateCmd.Flags().StringVar(&tierSlug, "slug", "", "Update tier slug")
	tiersUpdateCmd.Flags().StringVar(&tierDescription, "description", "", "Update description")
	tiersUpdateCmd.Flags().IntVar(&tierMonthlyPrice, "monthly-price", 0, "Update monthly price")
	tiersUpdateCmd.Flags().IntVar(&tierYearlyPrice, "yearly-price", 0, "Update yearly price")
	tiersUpdateCmd.Flags().StringVar(&tierActive, "active", "", "Set active status (true/false)")
	tiersUpdateCmd.Flags().StringVar(&tierWelcomePageURL, "welcome-page-url", "", "Set welcome page URL")
	tiersUpdateCmd.Flags().StringVar(&tierVisibility, "visibility", "", "Update visibility")
	tiersUpdateCmd.Flags().IntVar(&tierTrialDays, "trial-days", 0, "Update trial period")
}

type Tier struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Description      string `json:"description,omitempty"`
	Active           bool   `json:"active"`
	Type             string `json:"type"`
	WelcomePageURL   string `json:"welcome_page_url,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	Visibility       string `json:"visibility"`
	MonthlyPrice     int    `json:"monthly_price,omitempty"`
	YearlyPrice      int    `json:"yearly_price,omitempty"`
	Currency         string `json:"currency,omitempty"`
	TrialDays        int    `json:"trial_days"`
}

type tiersResponse struct {
	Tiers []Tier `json:"tiers"`
	Meta  struct {
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Pages int `json:"pages"`
			Total int `json:"total"`
			Next  int `json:"next"`
		} `json:"pagination"`
	} `json:"meta"`
}

func runTiersList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	data, err := client.Get("/tiers/", nil)
	if err != nil {
		return err
	}

	var resp tiersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Tiers)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tACTIVE\tVISIBILITY")
	for _, t := range resp.Tiers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", t.ID, t.Name, t.Type, t.Active, t.Visibility)
	}
	return w.Flush()
}

func runTiersGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	tier, err := getTier(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tier)
	}

	fmt.Printf("ID:          %s\n", tier.ID)
	fmt.Printf("Name:        %s\n", tier.Name)
	fmt.Printf("Slug:        %s\n", tier.Slug)
	fmt.Printf("Type:        %s\n", tier.Type)
	fmt.Printf("Active:      %v\n", tier.Active)
	fmt.Printf("Visibility:  %s\n", tier.Visibility)
	if tier.Description != "" {
		fmt.Printf("Description: %s\n", tier.Description)
	}
	if tier.MonthlyPrice > 0 {
		fmt.Printf("Monthly:     %d %s\n", tier.MonthlyPrice, tier.Currency)
	}
	if tier.YearlyPrice > 0 {
		fmt.Printf("Yearly:      %d %s\n", tier.YearlyPrice, tier.Currency)
	}
	if tier.TrialDays > 0 {
		fmt.Printf("Trial:       %d days\n", tier.TrialDays)
	}
	return nil
}

func runTiersCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	tier := map[string]interface{}{
		"name": args[0],
		"type": "paid",
	}

	if tierSlug != "" {
		tier["slug"] = tierSlug
	}
	if tierDescription != "" {
		tier["description"] = tierDescription
	}
	if tierMonthlyPrice > 0 {
		tier["monthly_price"] = tierMonthlyPrice
	}
	if tierYearlyPrice > 0 {
		tier["yearly_price"] = tierYearlyPrice
	}
	if tierCurrency != "" {
		tier["currency"] = tierCurrency
	}
	if tierVisibility != "" {
		tier["visibility"] = tierVisibility
	}
	if tierTrialDays > 0 {
		tier["trial_days"] = tierTrialDays
	}

	body := map[string]interface{}{
		"tiers": []interface{}{tier},
	}

	data, err := client.Post("/tiers/", body)
	if err != nil {
		return err
	}

	var resp tiersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tiers) == 0 {
		return fmt.Errorf("no tier in response")
	}

	created := resp.Tiers[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created tier: %s\n", created.Name)
	fmt.Printf("  ID:   %s\n", created.ID)
	fmt.Printf("  Slug: %s\n", created.Slug)
	return nil
}

func runTiersUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getTier(client, args[0])
	if err != nil {
		return err
	}

	tier := map[string]interface{}{}

	if tierSlug != "" {
		tier["slug"] = tierSlug
	}
	if tierDescription != "" {
		tier["description"] = tierDescription
	}
	if cmd.Flags().Changed("monthly-price") {
		tier["monthly_price"] = tierMonthlyPrice
	}
	if cmd.Flags().Changed("yearly-price") {
		tier["yearly_price"] = tierYearlyPrice
	}
	if tierActive != "" {
		tier["active"] = tierActive == "true"
	}
	if tierWelcomePageURL != "" {
		tier["welcome_page_url"] = tierWelcomePageURL
	}
	if tierVisibility != "" {
		tier["visibility"] = tierVisibility
	}
	if cmd.Flags().Changed("trial-days") {
		tier["trial_days"] = tierTrialDays
	}

	if len(tier) == 0 {
		return fmt.Errorf("no updates specified")
	}

	body := map[string]interface{}{
		"tiers": []interface{}{tier},
	}

	data, err := client.Put(fmt.Sprintf("/tiers/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp tiersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tiers) == 0 {
		return fmt.Errorf("no tier in response")
	}

	updated := resp.Tiers[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated tier: %s\n", updated.Name)
	fmt.Printf("  ID: %s\n", updated.ID)
	return nil
}

func getTier(client *api.Client, idOrSlug string) (*Tier, error) {
	data, err := client.Get(fmt.Sprintf("/tiers/%s/", idOrSlug), nil)
	if err == nil {
		var resp tiersResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Tiers) > 0 {
			return &resp.Tiers[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/tiers/", params)
	if err != nil {
		return nil, err
	}

	var resp tiersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tiers) == 0 {
		return nil, fmt.Errorf("tier not found: %s", idOrSlug)
	}

	return &resp.Tiers[0], nil
}
