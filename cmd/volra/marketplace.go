package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/romerox3/volra/internal/marketplace"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Discover and install community agent templates",
	Long:  "Search, list, and install agent templates from the Volra marketplace.",
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search marketplace templates",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)

		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		return runMarketplaceSearch(p, query)
	},
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all marketplace templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runMarketplaceSearch(p, "")
	},
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <template-name>",
	Short: "Install a marketplace template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runMarketplaceInstall(p, args[0])
	},
}

func init() {
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	rootCmd.AddCommand(marketplaceCmd)
}

func runMarketplaceSearch(p output.Presenter, query string) error {
	client := marketplace.NewClient()
	idx, err := client.FetchIndex()
	if err != nil {
		return &output.UserError{
			Code: output.CodeMarketplaceFetch,
			What: fmt.Sprintf("Could not fetch marketplace index: %v", err),
			Fix:  "Check your network connection and try again",
		}
	}

	results := marketplace.Search(idx, query)

	if len(results) == 0 {
		p.Result("No templates found")
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding results: %w", err)
		}
		p.Result(string(data))
		return nil
	}

	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "Name\tDescription\tFramework\tAuthor")
	for _, t := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Description, t.Framework, t.Author)
	}
	w.Flush()

	p.Result(string(buf))
	return nil
}

func runMarketplaceInstall(p output.Presenter, name string) error {
	client := marketplace.NewClient()
	idx, err := client.FetchIndex()
	if err != nil {
		return &output.UserError{
			Code: output.CodeMarketplaceFetch,
			What: fmt.Sprintf("Could not fetch marketplace index: %v", err),
			Fix:  "Check your network connection and try again",
		}
	}

	tmpl, ok := marketplace.Lookup(idx, name)
	if !ok {
		return &output.UserError{
			Code: output.CodeMarketplaceNotFound,
			What: fmt.Sprintf("Template %q not found in marketplace", name),
			Fix:  "Run 'volra marketplace list' to see available templates",
		}
	}

	p.Progress(fmt.Sprintf("Installing template %s...", tmpl.Name))

	if err := client.Install(tmpl, "."); err != nil {
		return fmt.Errorf("installing template: %w", err)
	}

	p.Result(fmt.Sprintf("Template %s installed successfully", tmpl.Name))
	return nil
}
