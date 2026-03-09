package main

import (
	"github.com/romerox3/volra/internal/dev"
	"github.com/romerox3/volra/internal/docker"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev [path]",
	Short: "Start development mode with hot-reload",
	Long:  "Watch project files and automatically rebuild the agent container on changes. Uses docker compose watch internally. Requires Docker Compose >= 2.22.0.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return dev.Run(cmd.Context(), dir, p, dr, nil, nil)
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}
