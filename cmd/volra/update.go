package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Volra to the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		client := &http.Client{Timeout: 30 * time.Second}
		return runUpdate(cmd, p, client)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, p output.Presenter, client update.HTTPClient) error {
	// Check if installed via Homebrew
	if update.IsHomebrew() {
		p.Warn(&output.UserWarning{
			What:     "Volra was installed via Homebrew",
			Assumed:  "Self-update is not available for Homebrew installations",
			Override: "Use `brew upgrade volra` instead",
		})
		return nil
	}

	p.Progress("Checking for updates...")

	release, err := update.CheckLatest(cmd.Context(), client)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if update.IsUpToDate(version, release.Version) {
		p.Result(fmt.Sprintf("Already up to date (%s)", version))
		return nil
	}

	p.Progress(fmt.Sprintf("Downloading %s...", release.Version))
	if err := update.Download(cmd.Context(), client, release); err != nil {
		return fmt.Errorf("updating: %w", err)
	}

	p.Result(fmt.Sprintf("Updated from %s to %s", version, release.Version))
	return nil
}
