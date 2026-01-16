package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/api"
	"github.com/teal-bauer/specter/internal/config"
	"github.com/teal-bauer/specter/internal/content"
)

var pagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Manage pages",
}

var pagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pages",
	RunE:  runPagesList,
}

var pagesGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a page by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runPagesGet,
}

var pagesCreateCmd = &cobra.Command{
	Use:   "create <file.md>",
	Short: "Create a page from a markdown file",
	Args:  cobra.ExactArgs(1),
	RunE:  runPagesCreate,
}

var pagesUpdateCmd = &cobra.Command{
	Use:   "update <id-or-slug> [file.md]",
	Short: "Update a page",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPagesUpdate,
}

var pagesDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-slug>",
	Short: "Delete a page",
	Args:  cobra.ExactArgs(1),
	RunE:  runPagesDelete,
}

var (
	pagesLimit  int
	pagesPage   int
	pagesAll    bool
	pagesStatus string
)

func init() {
	rootCmd.AddCommand(pagesCmd)
	pagesCmd.AddCommand(pagesListCmd)
	pagesCmd.AddCommand(pagesGetCmd)
	pagesCmd.AddCommand(pagesCreateCmd)
	pagesCmd.AddCommand(pagesUpdateCmd)
	pagesCmd.AddCommand(pagesDeleteCmd)

	pagesListCmd.Flags().IntVar(&pagesLimit, "limit", 15, "Number of pages to return")
	pagesListCmd.Flags().IntVar(&pagesPage, "page", 1, "Page number")
	pagesListCmd.Flags().BoolVar(&pagesAll, "all", false, "Fetch all pages")

	pagesCreateCmd.Flags().StringVar(&pagesStatus, "status", "", "Page status: draft or published")
	pagesUpdateCmd.Flags().StringVar(&pagesStatus, "status", "", "Update page status")
}

type Page struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	HTML        string `json:"html,omitempty"`
	Status      string `json:"status"`
	Featured    bool   `json:"featured"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	PublishedAt string `json:"published_at,omitempty"`
	URL         string `json:"url,omitempty"`
	FeatureImg  string `json:"feature_image,omitempty"`
	Tags        []Tag  `json:"tags,omitempty"`
}

type pagesResponse struct {
	Pages []Page `json:"pages"`
	Meta  struct {
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Pages int `json:"pages"`
			Total int `json:"total"`
			Next  int `json:"next"`
			Prev  int `json:"prev"`
		} `json:"pagination"`
	} `json:"meta"`
}

func runPagesList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	var allPages []Page

	if pagesAll {
		page := 1
		for {
			params := url.Values{}
			params.Set("limit", "100")
			params.Set("page", fmt.Sprintf("%d", page))

			data, err := client.Get("/pages/", params)
			if err != nil {
				return err
			}

			var resp pagesResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			allPages = append(allPages, resp.Pages...)

			if resp.Meta.Pagination.Next == 0 {
				break
			}
			page = resp.Meta.Pagination.Next
		}
	} else {
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", pagesLimit))
		params.Set("page", fmt.Sprintf("%d", pagesPage))

		data, err := client.Get("/pages/", params)
		if err != nil {
			return err
		}

		var resp pagesResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		allPages = resp.Pages
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allPages)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPUBLISHED")
	for _, p := range allPages {
		published := p.PublishedAt
		if published == "" {
			published = "-"
		} else if len(published) > 10 {
			published = published[:10]
		}
		title := p.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, title, p.Status, published)
	}
	return w.Flush()
}

func runPagesGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	page, err := getPage(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(page)
	}

	fmt.Printf("ID:        %s\n", page.ID)
	fmt.Printf("Title:     %s\n", page.Title)
	fmt.Printf("Slug:      %s\n", page.Slug)
	fmt.Printf("Status:    %s\n", page.Status)
	fmt.Printf("URL:       %s\n", page.URL)
	if page.PublishedAt != "" {
		fmt.Printf("Published: %s\n", page.PublishedAt)
	}
	if len(page.Tags) > 0 {
		var tagNames []string
		for _, t := range page.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Printf("Tags:      %s\n", strings.Join(tagNames, ", "))
	}
	return nil
}

func runPagesCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	parsed, err := content.ParseFile(args[0])
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	page := map[string]interface{}{
		"title": parsed.Frontmatter.Title,
		"html":  parsed.HTML,
	}

	if parsed.Frontmatter.Slug != "" {
		page["slug"] = parsed.Frontmatter.Slug
	}
	if parsed.Frontmatter.FeatureImg != "" {
		page["feature_image"] = parsed.Frontmatter.FeatureImg
	}
	if parsed.Frontmatter.Featured {
		page["featured"] = true
	}

	status := "draft"
	if parsed.Frontmatter.Status != "" {
		status = parsed.Frontmatter.Status
	}
	if pagesStatus != "" {
		status = pagesStatus
	}
	page["status"] = status

	if len(parsed.Frontmatter.Tags) > 0 {
		var tags []map[string]string
		for _, t := range parsed.Frontmatter.Tags {
			tags = append(tags, map[string]string{"name": t})
		}
		page["tags"] = tags
	}

	body := map[string]interface{}{
		"pages": []interface{}{page},
	}

	data, err := client.Post("/pages/", body)
	if err != nil {
		return err
	}

	var resp pagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Pages) == 0 {
		return fmt.Errorf("no page in response")
	}

	created := resp.Pages[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created page: %s\n", created.Title)
	fmt.Printf("  ID:     %s\n", created.ID)
	fmt.Printf("  Slug:   %s\n", created.Slug)
	fmt.Printf("  Status: %s\n", created.Status)
	return nil
}

func runPagesUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getPage(client, args[0])
	if err != nil {
		return err
	}

	page := map[string]interface{}{
		"updated_at": existing.UpdatedAt,
	}

	if len(args) > 1 {
		parsed, err := content.ParseFile(args[1])
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}

		if parsed.Frontmatter.Title != "" {
			page["title"] = parsed.Frontmatter.Title
		}
		page["html"] = parsed.HTML

		if parsed.Frontmatter.Slug != "" {
			page["slug"] = parsed.Frontmatter.Slug
		}
		if parsed.Frontmatter.FeatureImg != "" {
			page["feature_image"] = parsed.Frontmatter.FeatureImg
		}
		page["featured"] = parsed.Frontmatter.Featured

		if parsed.Frontmatter.Status != "" && pagesStatus == "" {
			page["status"] = parsed.Frontmatter.Status
		}

		if len(parsed.Frontmatter.Tags) > 0 {
			var tags []map[string]string
			for _, t := range parsed.Frontmatter.Tags {
				tags = append(tags, map[string]string{"name": t})
			}
			page["tags"] = tags
		}
	}

	if pagesStatus != "" {
		page["status"] = pagesStatus
	}

	body := map[string]interface{}{
		"pages": []interface{}{page},
	}

	data, err := client.Put(fmt.Sprintf("/pages/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp pagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Pages) == 0 {
		return fmt.Errorf("no page in response")
	}

	updated := resp.Pages[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated page: %s\n", updated.Title)
	fmt.Printf("  ID:     %s\n", updated.ID)
	fmt.Printf("  Status: %s\n", updated.Status)
	return nil
}

func runPagesDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getPage(client, args[0])
	if err != nil {
		return err
	}

	_, err = client.Delete(fmt.Sprintf("/pages/%s/", existing.ID))
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"deleted": existing.ID,
			"title":   existing.Title,
		})
	}

	fmt.Printf("Deleted page: %s (%s)\n", existing.Title, existing.ID)
	return nil
}

func getPage(client *api.Client, idOrSlug string) (*Page, error) {
	data, err := client.Get(fmt.Sprintf("/pages/%s/", idOrSlug), nil)
	if err == nil {
		var resp pagesResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Pages) > 0 {
			return &resp.Pages[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/pages/", params)
	if err != nil {
		return nil, err
	}

	var resp pagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Pages) == 0 {
		return nil, fmt.Errorf("page not found: %s", idOrSlug)
	}

	return &resp.Pages[0], nil
}
