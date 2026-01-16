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

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users (staff)",
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	RunE:  runUsersList,
}

var usersGetCmd = &cobra.Command{
	Use:   "get <id-or-slug>",
	Short: "Get a user by ID or slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersGet,
}

var usersLimit int

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersGetCmd)

	usersListCmd.Flags().IntVar(&usersLimit, "limit", 15, "Number of users to return")
}

type User struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Email            string `json:"email"`
	ProfileImage     string `json:"profile_image,omitempty"`
	CoverImage       string `json:"cover_image,omitempty"`
	Bio              string `json:"bio,omitempty"`
	Website          string `json:"website,omitempty"`
	Location         string `json:"location,omitempty"`
	Status           string `json:"status"`
	Accessibility    string `json:"accessibility,omitempty"`
	CreatedAt        string `json:"created_at"`
	LastSeen         string `json:"last_seen,omitempty"`
	URL              string `json:"url,omitempty"`
	Roles            []Role `json:"roles,omitempty"`
}

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type usersResponse struct {
	Users []User `json:"users"`
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

func runUsersList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", usersLimit))

	data, err := client.Get("/users/", params)
	if err != nil {
		return err
	}

	var resp usersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Users)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tSTATUS")
	for _, u := range resp.Users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Status)
	}
	return w.Flush()
}

func runUsersGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	user, err := getUser(client, args[0])
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(user)
	}

	printUser(*user)
	return nil
}

func printUser(u User) {
	fmt.Printf("ID:       %s\n", u.ID)
	fmt.Printf("Name:     %s\n", u.Name)
	fmt.Printf("Slug:     %s\n", u.Slug)
	fmt.Printf("Email:    %s\n", u.Email)
	fmt.Printf("Status:   %s\n", u.Status)
	if u.Bio != "" {
		fmt.Printf("Bio:      %s\n", u.Bio)
	}
	if u.Website != "" {
		fmt.Printf("Website:  %s\n", u.Website)
	}
	if u.Location != "" {
		fmt.Printf("Location: %s\n", u.Location)
	}
	if len(u.Roles) > 0 {
		fmt.Print("Roles:    ")
		for i, r := range u.Roles {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(r.Name)
		}
		fmt.Println()
	}
	if u.LastSeen != "" {
		fmt.Printf("Last Seen: %s\n", u.LastSeen)
	}
}

func getUser(client *api.Client, idOrSlug string) (*User, error) {
	data, err := client.Get(fmt.Sprintf("/users/%s/", idOrSlug), nil)
	if err == nil {
		var resp usersResponse
		if err := json.Unmarshal(data, &resp); err == nil && len(resp.Users) > 0 {
			return &resp.Users[0], nil
		}
	}

	params := url.Values{}
	params.Set("filter", fmt.Sprintf("slug:%s", idOrSlug))
	data, err = client.Get("/users/", params)
	if err != nil {
		return nil, err
	}

	var resp usersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Users) == 0 {
		return nil, fmt.Errorf("user not found: %s", idOrSlug)
	}

	return &resp.Users[0], nil
}
