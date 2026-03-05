package main

import (
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/status"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show health and status of deployed agent",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return status.Run(cmd.Context(), dir, p, dr)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
