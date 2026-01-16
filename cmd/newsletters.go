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

var newslettersCmd = &cobra.Command{
	Use:   "newsletters",
	Short: "Manage newsletters",
}

var newslettersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List newsletters",
	RunE:  runNewslettersList,
}

var newslettersGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a newsletter by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewslettersGet,
}

var newslettersCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a newsletter",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewslettersCreate,
}

var newslettersUpdateCmd = &cobra.Command{
	Use:   "update <id-or-slug>",
	Short: "Update a newsletter",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewslettersUpdate,
}

var (
	nlSlug           string
	nlDescription    string
	nlSenderName     string
	nlSenderEmail    string
	nlSenderReplyTo  string
	nlStatus         string
	nlSubscribeOnSignup string
	nlTitleFont      string
	nlBodyFont       string
	nlShowHeaderIcon string
	nlShowHeaderTitle string
	nlShowHeaderName string
)

func init() {
	rootCmd.AddCommand(newslettersCmd)
	newslettersCmd.AddCommand(newslettersListCmd)
	newslettersCmd.AddCommand(newslettersGetCmd)
	newslettersCmd.AddCommand(newslettersCreateCmd)
	newslettersCmd.AddCommand(newslettersUpdateCmd)

	newslettersCreateCmd.Flags().StringVar(&nlSlug, "slug", "", "Newsletter slug")
	newslettersCreateCmd.Flags().StringVar(&nlDescription, "description", "", "Newsletter description")
	newslettersCreateCmd.Flags().StringVar(&nlSenderName, "sender-name", "", "Sender name")
	newslettersCreateCmd.Flags().StringVar(&nlSenderEmail, "sender-email", "", "Sender email")
	newslettersCreateCmd.Flags().StringVar(&nlSenderReplyTo, "reply-to", "", "Reply-to address")

	newslettersUpdateCmd.Flags().StringVar(&nlSlug, "slug", "", "Update newsletter slug")
	newslettersUpdateCmd.Flags().StringVar(&nlDescription, "description", "", "Update description")
	newslettersUpdateCmd.Flags().StringVar(&nlSenderName, "sender-name", "", "Update sender name")
	newslettersUpdateCmd.Flags().StringVar(&nlSenderEmail, "sender-email", "", "Update sender email")
	newslettersUpdateCmd.Flags().StringVar(&nlSenderReplyTo, "reply-to", "", "Update reply-to")
	newslettersUpdateCmd.Flags().StringVar(&nlStatus, "status", "", "Update status (active/archived)")
	newslettersUpdateCmd.Flags().StringVar(&nlSubscribeOnSignup, "subscribe-on-signup", "", "Subscribe on signup (true/false)")
	newslettersUpdateCmd.Flags().StringVar(&nlTitleFont, "title-font", "", "Title font")
	newslettersUpdateCmd.Flags().StringVar(&nlBodyFont, "body-font", "", "Body font")
	newslettersUpdateCmd.Flags().StringVar(&nlShowHeaderIcon, "show-header-icon", "", "Show header icon (true/false)")
	newslettersUpdateCmd.Flags().StringVar(&nlShowHeaderTitle, "show-header-title", "", "Show header title (true/false)")
	newslettersUpdateCmd.Flags().StringVar(&nlShowHeaderName, "show-header-name", "", "Show header name (true/false)")
}

type Newsletter struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Description       string `json:"description,omitempty"`
	SenderName        string `json:"sender_name,omitempty"`
	SenderEmail       string `json:"sender_email,omitempty"`
	SenderReplyTo     string `json:"sender_reply_to,omitempty"`
	Status            string `json:"status"`
	Visibility        string `json:"visibility"`
	SubscribeOnSignup bool   `json:"subscribe_on_signup"`
	SortOrder         int    `json:"sort_order"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	TitleFont         string `json:"title_font_category,omitempty"`
	BodyFont          string `json:"body_font_category,omitempty"`
	ShowHeaderIcon    bool   `json:"show_header_icon"`
	ShowHeaderTitle   bool   `json:"show_header_title"`
	ShowHeaderName    bool   `json:"show_header_name"`
}

type newslettersResponse struct {
	Newsletters []Newsletter `json:"newsletters"`
	Meta        struct {
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Pages int `json:"pages"`
			Total int `json:"total"`
			Next  int `json:"next"`
		} `json:"pagination"`
	} `json:"meta"`
}

func runNewslettersList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	data, err := client.Get("/newsletters/", nil)
	if err != nil {
		return err
	}

	var resp newslettersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Newsletters)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tSUBSCRIBE ON SIGNUP")
	for _, n := range resp.Newsletters {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", n.ID, n.Name, n.Status, n.SubscribeOnSignup)
	}
	return w.Flush()
}

func runNewslettersGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	nl, err := getNewsletter(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(nl)
	}

	fmt.Printf("ID:               %s\n", nl.ID)
	fmt.Printf("Name:             %s\n", nl.Name)
	fmt.Printf("Slug:             %s\n", nl.Slug)
	fmt.Printf("Status:           %s\n", nl.Status)
	if nl.Description != "" {
		fmt.Printf("Description:      %s\n", nl.Description)
	}
	if nl.SenderName != "" {
		fmt.Printf("Sender Name:      %s\n", nl.SenderName)
	}
	if nl.SenderEmail != "" {
		fmt.Printf("Sender Email:     %s\n", nl.SenderEmail)
	}
	fmt.Printf("Subscribe Signup: %v\n", nl.SubscribeOnSignup)
	return nil
}

func runNewslettersCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	nl := map[string]interface{}{
		"name": args[0],
	}

	if nlSlug != "" {
		nl["slug"] = nlSlug
	}
	if nlDescription != "" {
		nl["description"] = nlDescription
	}
	if nlSenderName != "" {
		nl["sender_name"] = nlSenderName
	}
	if nlSenderEmail != "" {
		nl["sender_email"] = nlSenderEmail
	}
	if nlSenderReplyTo != "" {
		nl["sender_reply_to"] = nlSenderReplyTo
	}

	body := map[string]interface{}{
		"newsletters": []interface{}{nl},
	}

	data, err := client.Post("/newsletters/", body)
	if err != nil {
		return err
	}

	var resp newslettersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Newsletters) == 0 {
		return fmt.Errorf("no newsletter in response")
	}

	created := resp.Newsletters[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created newsletter: %s\n", created.Name)
	fmt.Printf("  ID:   %s\n", created.ID)
	fmt.Printf("  Slug: %s\n", created.Slug)
	return nil
}

func runNewslettersUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getNewsletter(client, args[0])
	if err != nil {
		return err
	}

	nl := map[string]interface{}{}

	if nlSlug != "" {
		nl["slug"] = nlSlug
	}
	if nlDescription != "" {
		nl["description"] = nlDescription
	}
	if nlSenderName != "" {
		nl["sender_name"] = nlSenderName
	}
	if nlSenderEmail != "" {
		nl["sender_email"] = nlSenderEmail
	}
	if nlSenderReplyTo != "" {
		nl["sender_reply_to"] = nlSenderReplyTo
	}
	if nlStatus != "" {
		nl["status"] = nlStatus
	}
	if nlSubscribeOnSignup != "" {
		nl["subscribe_on_signup"] = nlSubscribeOnSignup == "true"
	}
	if nlTitleFont != "" {
		nl["title_font_category"] = nlTitleFont
	}
	if nlBodyFont != "" {
		nl["body_font_category"] = nlBodyFont
	}
	if nlShowHeaderIcon != "" {
		nl["show_header_icon"] = nlShowHeaderIcon == "true"
	}
	if nlShowHeaderTitle != "" {
		nl["show_header_title"] = nlShowHeaderTitle == "true"
	}
	if nlShowHeaderName != "" {
		nl["show_header_name"] = nlShowHeaderName == "true"
	}

	if len(nl) == 0 {
		return fmt.Errorf("no updates specified")
	}

	body := map[string]interface{}{
		"newsletters": []interface{}{nl},
	}

	data, err := client.Put(fmt.Sprintf("/newsletters/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp newslettersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Newsletters) == 0 {
		return fmt.Errorf("no newsletter in response")
	}

	updated := resp.Newsletters[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated newsletter: %s\n", updated.Name)
	fmt.Printf("  ID: %s\n", updated.ID)
	return nil
}

func getNewsletter(client *api.Client, idOrSlug string) (*Newsletter, error) {
	data, err := client.Get(fmt.Sprintf("/newsletters/%s/", idOrSlug), nil)
	if err == nil {
		var resp newslettersResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Newsletters) > 0 {
			return &resp.Newsletters[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/newsletters/", params)
	if err != nil {
		return nil, err
	}

	var resp newslettersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Newsletters) == 0 {
		return nil, fmt.Errorf("newsletter not found: %s", idOrSlug)
	}

	return &resp.Newsletters[0], nil
}
