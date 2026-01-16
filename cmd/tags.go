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

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage tags",
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags",
	RunE:  runTagsList,
}

var tagsGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a tag by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagsGet,
}

var tagsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagsCreate,
}

var tagsUpdateCmd = &cobra.Command{
	Use:   "update <id-or-slug>",
	Short: "Update a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagsUpdate,
}

var tagsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-slug>",
	Short: "Delete a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagsDelete,
}

var (
	tagsLimit       int
	tagsAll         bool
	tagSlug         string
	tagDescription  string
	tagFeatureImage string
	tagVisibility   string
	tagMetaTitle    string
	tagMetaDesc     string
)

func init() {
	rootCmd.AddCommand(tagsCmd)
	tagsCmd.AddCommand(tagsListCmd)
	tagsCmd.AddCommand(tagsGetCmd)
	tagsCmd.AddCommand(tagsCreateCmd)
	tagsCmd.AddCommand(tagsUpdateCmd)
	tagsCmd.AddCommand(tagsDeleteCmd)

	tagsListCmd.Flags().IntVar(&tagsLimit, "limit", 15, "Number of tags to return")
	tagsListCmd.Flags().BoolVar(&tagsAll, "all", false, "Fetch all tags")

	tagsCreateCmd.Flags().StringVar(&tagSlug, "slug", "", "Tag slug")
	tagsCreateCmd.Flags().StringVar(&tagDescription, "description", "", "Tag description")
	tagsCreateCmd.Flags().StringVar(&tagFeatureImage, "feature-image", "", "Feature image URL")
	tagsCreateCmd.Flags().StringVar(&tagVisibility, "visibility", "public", "Visibility: public or internal")

	tagsUpdateCmd.Flags().StringVar(&tagSlug, "slug", "", "Update tag slug")
	tagsUpdateCmd.Flags().StringVar(&tagDescription, "description", "", "Update description")
	tagsUpdateCmd.Flags().StringVar(&tagFeatureImage, "feature-image", "", "Update feature image URL")
	tagsUpdateCmd.Flags().StringVar(&tagVisibility, "visibility", "", "Update visibility")
	tagsUpdateCmd.Flags().StringVar(&tagMetaTitle, "meta-title", "", "Update meta title")
	tagsUpdateCmd.Flags().StringVar(&tagMetaDesc, "meta-description", "", "Update meta description")
}

type Tag struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	FeatureImage string `json:"feature_image,omitempty"`
	Visibility   string `json:"visibility"`
	MetaTitle    string `json:"meta_title,omitempty"`
	MetaDesc     string `json:"meta_description,omitempty"`
	URL          string `json:"url,omitempty"`
	PostCount    int    `json:"count,omitempty"`
}

type tagsResponse struct {
	Tags []Tag `json:"tags"`
	Meta struct {
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Pages int `json:"pages"`
			Total int `json:"total"`
			Next  int `json:"next"`
		} `json:"pagination"`
	} `json:"meta"`
}

func runTagsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	var allTags []Tag

	if tagsAll {
		page := 1
		for {
			params := url.Values{}
			params.Set("limit", "100")
			params.Set("page", fmt.Sprintf("%d", page))

			data, err := client.Get("/tags/", params)
			if err != nil {
				return err
			}

			var resp tagsResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			allTags = append(allTags, resp.Tags...)

			if resp.Meta.Pagination.Next == 0 {
				break
			}
			page = resp.Meta.Pagination.Next
		}
	} else {
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", tagsLimit))

		data, err := client.Get("/tags/", params)
		if err != nil {
			return err
		}

		var resp tagsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		allTags = resp.Tags
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allTags)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tVISIBILITY")
	for _, t := range allTags {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Name, t.Slug, t.Visibility)
	}
	return w.Flush()
}

func runTagsGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	tag, err := getTag(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tag)
	}

	fmt.Printf("ID:          %s\n", tag.ID)
	fmt.Printf("Name:        %s\n", tag.Name)
	fmt.Printf("Slug:        %s\n", tag.Slug)
	fmt.Printf("Visibility:  %s\n", tag.Visibility)
	if tag.Description != "" {
		fmt.Printf("Description: %s\n", tag.Description)
	}
	if tag.URL != "" {
		fmt.Printf("URL:         %s\n", tag.URL)
	}
	return nil
}

func runTagsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	tag := map[string]interface{}{
		"name": args[0],
	}

	if tagSlug != "" {
		tag["slug"] = tagSlug
	}
	if tagDescription != "" {
		tag["description"] = tagDescription
	}
	if tagFeatureImage != "" {
		tag["feature_image"] = tagFeatureImage
	}
	if tagVisibility != "" {
		tag["visibility"] = tagVisibility
	}

	body := map[string]interface{}{
		"tags": []interface{}{tag},
	}

	data, err := client.Post("/tags/", body)
	if err != nil {
		return err
	}

	var resp tagsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tags) == 0 {
		return fmt.Errorf("no tag in response")
	}

	created := resp.Tags[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created tag: %s\n", created.Name)
	fmt.Printf("  ID:   %s\n", created.ID)
	fmt.Printf("  Slug: %s\n", created.Slug)
	return nil
}

func runTagsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getTag(client, args[0])
	if err != nil {
		return err
	}

	tag := map[string]interface{}{}

	if tagSlug != "" {
		tag["slug"] = tagSlug
	}
	if tagDescription != "" {
		tag["description"] = tagDescription
	}
	if tagFeatureImage != "" {
		tag["feature_image"] = tagFeatureImage
	}
	if tagVisibility != "" {
		tag["visibility"] = tagVisibility
	}
	if tagMetaTitle != "" {
		tag["meta_title"] = tagMetaTitle
	}
	if tagMetaDesc != "" {
		tag["meta_description"] = tagMetaDesc
	}

	if len(tag) == 0 {
		return fmt.Errorf("no updates specified")
	}

	body := map[string]interface{}{
		"tags": []interface{}{tag},
	}

	data, err := client.Put(fmt.Sprintf("/tags/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp tagsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tags) == 0 {
		return fmt.Errorf("no tag in response")
	}

	updated := resp.Tags[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated tag: %s\n", updated.Name)
	fmt.Printf("  ID:   %s\n", updated.ID)
	fmt.Printf("  Slug: %s\n", updated.Slug)
	return nil
}

func runTagsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getTag(client, args[0])
	if err != nil {
		return err
	}

	_, err = client.Delete(fmt.Sprintf("/tags/%s/", existing.ID))
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"deleted": existing.ID,
			"name":    existing.Name,
		})
	}

	fmt.Printf("Deleted tag: %s (%s)\n", existing.Name, existing.ID)
	return nil
}

func getTag(client *api.Client, idOrSlug string) (*Tag, error) {
	data, err := client.Get(fmt.Sprintf("/tags/%s/", idOrSlug), nil)
	if err == nil {
		var resp tagsResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Tags) > 0 {
			return &resp.Tags[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/tags/", params)
	if err != nil {
		return nil, err
	}

	var resp tagsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Tags) == 0 {
		return nil, fmt.Errorf("tag not found: %s", idOrSlug)
	}

	return &resp.Tags[0], nil
}
