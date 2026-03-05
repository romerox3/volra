package main

import (
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:     "volra",
	Short:   "Deploy AI agents to production with monitoring",
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output structured JSON instead of human-readable text")
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}

// newPresenter creates a Presenter based on the --json flag.
func newPresenter() output.Presenter {
	if jsonOutput {
		return output.NewPresenter(output.ModeJSON)
	}
	return output.NewPresenter(output.DetectMode())
}

// flushPresenter calls Flush on JSONPresenter if applicable.
func flushPresenter(p output.Presenter) {
	if jp, ok := p.(*output.JSONPresenter); ok {
		jp.Flush()
	}
}
