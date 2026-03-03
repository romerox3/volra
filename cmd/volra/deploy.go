package main

import (
	"github.com/antonioromero/volra/internal/deploy"
	"github.com/antonioromero/volra/internal/docker"
	"github.com/antonioromero/volra/internal/output"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [path]",
	Short: "Deploy agent with monitoring stack",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := output.NewPresenter(output.DetectMode())
		dr := docker.NewExecRunner()
		return deploy.Run(cmd.Context(), dir, p, dr)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
