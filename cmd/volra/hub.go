package main

import (
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/hub"
	"github.com/spf13/cobra"
)

var hubCmd = &cobra.Command{
	Use:   "hub",
	Short: "Manage unified multi-agent dashboard",
	Long:  "Start, stop, or check the status of the unified Grafana dashboard that shows all registered agents.",
}

var hubStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start unified dashboard",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return hub.Start(cmd.Context(), p, dr)
	},
}

var hubStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop unified dashboard",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return hub.Stop(cmd.Context(), p, dr)
	},
}

var hubStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show registered agents and hub state",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return hub.Status(cmd.Context(), p, dr)
	},
}

func init() {
	hubCmd.AddCommand(hubStartCmd)
	hubCmd.AddCommand(hubStopCmd)
	hubCmd.AddCommand(hubStatusCmd)
	rootCmd.AddCommand(hubCmd)
}
