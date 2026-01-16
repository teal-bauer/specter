package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/internal/config"
)

var profilesCmd = &cobra.Command{
	Use:     "profiles",
	Aliases: []string{"profile"},
	Short:   "List configured profiles",
	RunE:    runProfilesList,
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}

func runProfilesList(cmd *cobra.Command, args []string) error {
	names, defaultName, err := config.ListInstances()
	if err != nil {
		return fmt.Errorf("no profiles configured (run 'specter login' to set up)")
	}

	if len(names) == 0 {
		fmt.Println("No profiles configured. Run 'specter login' to set up.")
		return nil
	}

	if config.OutputFormat() == "json" {
		result := map[string]any{
			"profiles": names,
			"default":  defaultName,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tDEFAULT")
	for _, name := range names {
		isDefault := ""
		if name == defaultName {
			isDefault = "*"
		}
		fmt.Fprintf(w, "%s\t%s\n", name, isDefault)
	}
	return w.Flush()
}
