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

var postsCmd = &cobra.Command{
	Use:   "posts",
	Short: "Manage posts",
}

var postsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List posts",
	RunE:  runPostsList,
}

var postsGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a post by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runPostsGet,
}

var postsCreateCmd = &cobra.Command{
	Use:   "create <file.md>",
	Short: "Create a post from a markdown file",
	Long:  "Create a post from a markdown file with YAML frontmatter. Use '-' to read from stdin.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPostsCreate,
}

var postsUpdateCmd = &cobra.Command{
	Use:   "update <id-or-slug> [file.md]",
	Short: "Update a post",
	Long:  "Update a post. Provide a markdown file to update content, or use flags to update metadata only.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPostsUpdate,
}

var postsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-slug>",
	Short: "Delete a post",
	Args:  cobra.ExactArgs(1),
	RunE:  runPostsDelete,
}

// Flag variables
var (
	postsLimit     int
	postsPage      int
	postsAll       bool
	postsStatus    string
	postsPublishAt string
)

func init() {
	rootCmd.AddCommand(postsCmd)
	postsCmd.AddCommand(postsListCmd)
	postsCmd.AddCommand(postsGetCmd)
	postsCmd.AddCommand(postsCreateCmd)
	postsCmd.AddCommand(postsUpdateCmd)
	postsCmd.AddCommand(postsDeleteCmd)

	postsListCmd.Flags().IntVar(&postsLimit, "limit", 15, "Number of posts to return")
	postsListCmd.Flags().IntVar(&postsPage, "page", 1, "Page number")
	postsListCmd.Flags().BoolVar(&postsAll, "all", false, "Fetch all posts (ignores limit/page)")

	postsCreateCmd.Flags().StringVar(&postsStatus, "status", "", "Post status: draft, published, or scheduled")
	postsCreateCmd.Flags().StringVar(&postsPublishAt, "publish-at", "", "Scheduled publish time (ISO 8601)")

	postsUpdateCmd.Flags().StringVar(&postsStatus, "status", "", "Update post status")
	postsUpdateCmd.Flags().StringVar(&postsPublishAt, "publish-at", "", "Scheduled publish time (ISO 8601)")
}

// Post represents a Ghost post
type Post struct {
	ID          string   `json:"id"`
	UUID        string   `json:"uuid"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	HTML        string   `json:"html,omitempty"`
	Status      string   `json:"status"`
	Visibility  string   `json:"visibility"`
	Featured    bool     `json:"featured"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	PublishedAt string   `json:"published_at,omitempty"`
	Excerpt     string   `json:"excerpt,omitempty"`
	Tags        []Tag    `json:"tags,omitempty"`
	URL         string   `json:"url,omitempty"`
	FeatureImg  string   `json:"feature_image,omitempty"`
	MetaTitle   string   `json:"meta_title,omitempty"`
	MetaDesc    string   `json:"meta_description,omitempty"`
}

type postsResponse struct {
	Posts []Post `json:"posts"`
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

func runPostsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	var allPosts []Post

	if postsAll {
		page := 1
		for {
			params := url.Values{}
			params.Set("limit", "100")
			params.Set("page", fmt.Sprintf("%d", page))

			data, err := client.Get("/posts/", params)
			if err != nil {
				return err
			}

			var resp postsResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			allPosts = append(allPosts, resp.Posts...)

			if resp.Meta.Pagination.Next == 0 {
				break
			}
			page = resp.Meta.Pagination.Next
		}
	} else {
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", postsLimit))
		params.Set("page", fmt.Sprintf("%d", postsPage))

		data, err := client.Get("/posts/", params)
		if err != nil {
			return err
		}

		var resp postsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		allPosts = resp.Posts
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allPosts)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPUBLISHED")
	for _, p := range allPosts {
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

func runPostsGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	idOrSlug := args[0]
	path := fmt.Sprintf("/posts/%s/", idOrSlug)

	// Try by ID first, then by slug
	data, err := client.Get(path, nil)
	if err != nil {
		// Try by slug
		params := url.Values{}
		params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
		data, err = client.Get("/posts/", params)
		if err != nil {
			return err
		}

		var resp postsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		if len(resp.Posts) == 0 {
			return fmt.Errorf("post not found: %s", idOrSlug)
		}

		if config.OutputFormat() == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp.Posts[0])
		}

		printPost(resp.Posts[0])
		return nil
	}

	var resp postsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Posts) == 0 {
		return fmt.Errorf("post not found: %s", idOrSlug)
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Posts[0])
	}

	printPost(resp.Posts[0])
	return nil
}

func printPost(p Post) {
	fmt.Printf("ID:        %s\n", p.ID)
	fmt.Printf("Title:     %s\n", p.Title)
	fmt.Printf("Slug:      %s\n", p.Slug)
	fmt.Printf("Status:    %s\n", p.Status)
	fmt.Printf("URL:       %s\n", p.URL)
	if p.PublishedAt != "" {
		fmt.Printf("Published: %s\n", p.PublishedAt)
	}
	if len(p.Tags) > 0 {
		var tagNames []string
		for _, t := range p.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Printf("Tags:      %s\n", strings.Join(tagNames, ", "))
	}
	if p.Excerpt != "" {
		fmt.Printf("Excerpt:   %s\n", p.Excerpt)
	}
}

func runPostsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	parsed, err := content.ParseFile(args[0])
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	post := map[string]interface{}{
		"title": parsed.Frontmatter.Title,
		"html":  parsed.HTML,
	}

	if parsed.Frontmatter.Slug != "" {
		post["slug"] = parsed.Frontmatter.Slug
	}
	if parsed.Frontmatter.Excerpt != "" {
		post["custom_excerpt"] = parsed.Frontmatter.Excerpt
	}
	if parsed.Frontmatter.MetaTitle != "" {
		post["meta_title"] = parsed.Frontmatter.MetaTitle
	}
	if parsed.Frontmatter.MetaDesc != "" {
		post["meta_description"] = parsed.Frontmatter.MetaDesc
	}
	if parsed.Frontmatter.FeatureImg != "" {
		post["feature_image"] = parsed.Frontmatter.FeatureImg
	}
	if parsed.Frontmatter.Featured {
		post["featured"] = true
	}

	// Status priority: CLI flag > frontmatter > default (draft)
	status := "draft"
	if parsed.Frontmatter.Status != "" {
		status = parsed.Frontmatter.Status
	}
	if postsStatus != "" {
		status = postsStatus
	}
	post["status"] = status

	if postsPublishAt != "" {
		post["published_at"] = postsPublishAt
	} else if parsed.Frontmatter.PublishedAt != "" {
		post["published_at"] = parsed.Frontmatter.PublishedAt
	}

	// Handle tags
	if len(parsed.Frontmatter.Tags) > 0 {
		var tags []map[string]string
		for _, t := range parsed.Frontmatter.Tags {
			tags = append(tags, map[string]string{"name": t})
		}
		post["tags"] = tags
	}

	body := map[string]interface{}{
		"posts": []interface{}{post},
	}

	data, err := client.Post("/posts/", body)
	if err != nil {
		return err
	}

	var resp postsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Posts) == 0 {
		return fmt.Errorf("no post in response")
	}

	created := resp.Posts[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created post: %s\n", created.Title)
	fmt.Printf("  ID:     %s\n", created.ID)
	fmt.Printf("  Slug:   %s\n", created.Slug)
	fmt.Printf("  Status: %s\n", created.Status)
	fmt.Printf("  URL:    %s\n", created.URL)
	return nil
}

func runPostsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	idOrSlug := args[0]

	// First, get the existing post to get its ID and updated_at
	existing, err := getPost(client, idOrSlug)
	if err != nil {
		return err
	}

	post := map[string]interface{}{
		"updated_at": existing.UpdatedAt,
	}

	// If a file is provided, update content
	if len(args) > 1 {
		parsed, err := content.ParseFile(args[1])
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}

		if parsed.Frontmatter.Title != "" {
			post["title"] = parsed.Frontmatter.Title
		}
		post["html"] = parsed.HTML

		if parsed.Frontmatter.Slug != "" {
			post["slug"] = parsed.Frontmatter.Slug
		}
		if parsed.Frontmatter.Excerpt != "" {
			post["custom_excerpt"] = parsed.Frontmatter.Excerpt
		}
		if parsed.Frontmatter.MetaTitle != "" {
			post["meta_title"] = parsed.Frontmatter.MetaTitle
		}
		if parsed.Frontmatter.MetaDesc != "" {
			post["meta_description"] = parsed.Frontmatter.MetaDesc
		}
		if parsed.Frontmatter.FeatureImg != "" {
			post["feature_image"] = parsed.Frontmatter.FeatureImg
		}
		post["featured"] = parsed.Frontmatter.Featured

		if parsed.Frontmatter.Status != "" && postsStatus == "" {
			post["status"] = parsed.Frontmatter.Status
		}

		if len(parsed.Frontmatter.Tags) > 0 {
			var tags []map[string]string
			for _, t := range parsed.Frontmatter.Tags {
				tags = append(tags, map[string]string{"name": t})
			}
			post["tags"] = tags
		}
	}

	// CLI flags override everything
	if postsStatus != "" {
		post["status"] = postsStatus
	}
	if postsPublishAt != "" {
		post["published_at"] = postsPublishAt
	}

	body := map[string]interface{}{
		"posts": []interface{}{post},
	}

	data, err := client.Put(fmt.Sprintf("/posts/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp postsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Posts) == 0 {
		return fmt.Errorf("no post in response")
	}

	updated := resp.Posts[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated post: %s\n", updated.Title)
	fmt.Printf("  ID:     %s\n", updated.ID)
	fmt.Printf("  Status: %s\n", updated.Status)
	return nil
}

func runPostsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	idOrSlug := args[0]

	// Get the post first to confirm and get the ID
	existing, err := getPost(client, idOrSlug)
	if err != nil {
		return err
	}

	_, err = client.Delete(fmt.Sprintf("/posts/%s/", existing.ID))
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"deleted": existing.ID,
			"title":   existing.Title,
		})
	}

	fmt.Printf("Deleted post: %s (%s)\n", existing.Title, existing.ID)
	return nil
}

func getPost(client *api.Client, idOrSlug string) (*Post, error) {
	// Try by ID first
	data, err := client.Get(fmt.Sprintf("/posts/%s/", idOrSlug), nil)
	if err == nil {
		var resp postsResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Posts) > 0 {
			return &resp.Posts[0], nil
		}
	}

	// Try by slug
	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/posts/", params)
	if err != nil {
		return nil, err
	}

	var resp postsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Posts) == 0 {
		return nil, fmt.Errorf("post not found: %s", idOrSlug)
	}

	return &resp.Posts[0], nil
}
