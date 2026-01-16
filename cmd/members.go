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

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage members",
}

var membersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List members",
	RunE:  runMembersList,
}

var membersGetCmd = &cobra.Command{
	Use:   "get <id-or-email>",
	Short: "Get a member by ID or email",
	Args:  cobra.ExactArgs(1),
	RunE:  runMembersGet,
}

var membersCreateCmd = &cobra.Command{
	Use:   "create <email>",
	Short: "Create a member",
	Args:  cobra.ExactArgs(1),
	RunE:  runMembersCreate,
}

var membersUpdateCmd = &cobra.Command{
	Use:   "update <id-or-email>",
	Short: "Update a member",
	Args:  cobra.ExactArgs(1),
	RunE:  runMembersUpdate,
}

var membersDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-email>",
	Short: "Delete a member",
	Args:  cobra.ExactArgs(1),
	RunE:  runMembersDelete,
}

var (
	membersLimit    int
	membersAll      bool
	membersFilter   string
	memberName      string
	memberNote      string
	memberLabels    []string
	memberNewsletter bool
)

func init() {
	rootCmd.AddCommand(membersCmd)
	membersCmd.AddCommand(membersListCmd)
	membersCmd.AddCommand(membersGetCmd)
	membersCmd.AddCommand(membersCreateCmd)
	membersCmd.AddCommand(membersUpdateCmd)
	membersCmd.AddCommand(membersDeleteCmd)

	membersListCmd.Flags().IntVar(&membersLimit, "limit", 15, "Number of members to return")
	membersListCmd.Flags().BoolVar(&membersAll, "all", false, "Fetch all members")
	membersListCmd.Flags().StringVar(&membersFilter, "filter", "", "Filter members (e.g., 'status:free')")

	membersCreateCmd.Flags().StringVar(&memberName, "name", "", "Member name")
	membersCreateCmd.Flags().StringVar(&memberNote, "note", "", "Member note")
	membersCreateCmd.Flags().StringSliceVar(&memberLabels, "labels", nil, "Member labels")
	membersCreateCmd.Flags().BoolVar(&memberNewsletter, "newsletter", true, "Subscribe to newsletter")

	membersUpdateCmd.Flags().StringVar(&memberName, "name", "", "Update member name")
	membersUpdateCmd.Flags().StringVar(&memberNote, "note", "", "Update member note")
	membersUpdateCmd.Flags().StringSliceVar(&memberLabels, "labels", nil, "Update member labels")
}

type Member struct {
	ID            string   `json:"id"`
	UUID          string   `json:"uuid"`
	Email         string   `json:"email"`
	Name          string   `json:"name,omitempty"`
	Note          string   `json:"note,omitempty"`
	Status        string   `json:"status"`
	Subscribed    bool     `json:"subscribed"`
	CreatedAt     string   `json:"created_at"`
	Labels        []Label  `json:"labels,omitempty"`
	Newsletters   []Newsletter `json:"newsletters,omitempty"`
}

type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type membersResponse struct {
	Members []Member `json:"members"`
	Meta    struct {
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Pages int `json:"pages"`
			Total int `json:"total"`
			Next  int `json:"next"`
		} `json:"pagination"`
	} `json:"meta"`
}

func runMembersList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	var allMembers []Member

	if membersAll {
		page := 1
		for {
			params := url.Values{}
			params.Set("limit", "100")
			params.Set("page", fmt.Sprintf("%d", page))
			if membersFilter != "" {
				params.Set("filter", membersFilter)
			}

			data, err := client.Get("/members/", params)
			if err != nil {
				return err
			}

			var resp membersResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			allMembers = append(allMembers, resp.Members...)

			if resp.Meta.Pagination.Next == 0 {
				break
			}
			page = resp.Meta.Pagination.Next
		}
	} else {
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", membersLimit))
		if membersFilter != "" {
			params.Set("filter", membersFilter)
		}

		data, err := client.Get("/members/", params)
		if err != nil {
			return err
		}

		var resp membersResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		allMembers = resp.Members
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allMembers)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tNAME\tSTATUS")
	for _, m := range allMembers {
		name := m.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.Email, name, m.Status)
	}
	return w.Flush()
}

func runMembersGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	member, err := getMember(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("ID:         %s\n", member.ID)
	fmt.Printf("Email:      %s\n", member.Email)
	if member.Name != "" {
		fmt.Printf("Name:       %s\n", member.Name)
	}
	fmt.Printf("Status:     %s\n", member.Status)
	fmt.Printf("Subscribed: %v\n", member.Subscribed)
	fmt.Printf("Created:    %s\n", member.CreatedAt)
	if member.Note != "" {
		fmt.Printf("Note:       %s\n", member.Note)
	}
	if len(member.Labels) > 0 {
		fmt.Print("Labels:     ")
		for i, l := range member.Labels {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(l.Name)
		}
		fmt.Println()
	}
	return nil
}

func runMembersCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	member := map[string]interface{}{
		"email": args[0],
	}

	if memberName != "" {
		member["name"] = memberName
	}
	if memberNote != "" {
		member["note"] = memberNote
	}
	if len(memberLabels) > 0 {
		var labels []map[string]string
		for _, l := range memberLabels {
			labels = append(labels, map[string]string{"name": l})
		}
		member["labels"] = labels
	}

	body := map[string]interface{}{
		"members": []interface{}{member},
	}

	data, err := client.Post("/members/", body)
	if err != nil {
		return err
	}

	var resp membersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Members) == 0 {
		return fmt.Errorf("no member in response")
	}

	created := resp.Members[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	}

	fmt.Printf("Created member: %s\n", created.Email)
	fmt.Printf("  ID:     %s\n", created.ID)
	fmt.Printf("  Status: %s\n", created.Status)
	return nil
}

func runMembersUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getMember(client, args[0])
	if err != nil {
		return err
	}

	member := map[string]interface{}{}

	if memberName != "" {
		member["name"] = memberName
	}
	if memberNote != "" {
		member["note"] = memberNote
	}
	if len(memberLabels) > 0 {
		var labels []map[string]string
		for _, l := range memberLabels {
			labels = append(labels, map[string]string{"name": l})
		}
		member["labels"] = labels
	}

	if len(member) == 0 {
		return fmt.Errorf("no updates specified")
	}

	body := map[string]interface{}{
		"members": []interface{}{member},
	}

	data, err := client.Put(fmt.Sprintf("/members/%s/", existing.ID), body)
	if err != nil {
		return err
	}

	var resp membersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Members) == 0 {
		return fmt.Errorf("no member in response")
	}

	updated := resp.Members[0]

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	}

	fmt.Printf("Updated member: %s\n", updated.Email)
	fmt.Printf("  ID: %s\n", updated.ID)
	return nil
}

func runMembersDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	existing, err := getMember(client, args[0])
	if err != nil {
		return err
	}

	_, err = client.Delete(fmt.Sprintf("/members/%s/", existing.ID))
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"deleted": existing.ID,
			"email":   existing.Email,
		})
	}

	fmt.Printf("Deleted member: %s (%s)\n", existing.Email, existing.ID)
	return nil
}

func getMember(client *api.Client, idOrEmail string) (*Member, error) {
	data, err := client.Get(fmt.Sprintf("/members/%s/", idOrEmail), nil)
	if err == nil {
		var resp membersResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Members) > 0 {
			return &resp.Members[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("email:%s", idOrEmail))
	data, err = client.Get("/members/", params)
	if err != nil {
		return nil, err
	}

	var resp membersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Members) == 0 {
		return nil, fmt.Errorf("member not found: %s", idOrEmail)
	}

	return &resp.Members[0], nil
}
