package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
	"github.com/spf13/cobra"
)

var removeVolumes bool

var downCmd = &cobra.Command{
	Use:   "down [path]",
	Short: "Stop all deployed services",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := newPresenter()
		defer flushPresenter(p)
		dr := docker.NewExecRunner()
		return runDown(cmd.Context(), dir, p, dr)
	},
}

func init() {
	downCmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "Also remove named volumes (destroys monitoring data)")
	rootCmd.AddCommand(downCmd)
}

func runDown(ctx context.Context, dir string, p output.Presenter, dr docker.DockerRunner) error {
	// Check if .volra/docker-compose.yml exists
	composePath := filepath.Join(dir, deploy.OutputDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return &output.UserError{
			Code: output.CodeNoDeploymentFound,
			What: "No deployment found",
			Fix:  "Run `volra deploy` first to create a deployment",
		}
	}

	// Load Agentfile for project name
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return err
	}

	// Build compose down args
	downArgs := []string{"compose", "-f", composePath, "-p", af.Name, "down"}
	if removeVolumes {
		downArgs = append(downArgs, "-v")
	}

	_, err = dr.Run(ctx, downArgs...)
	if err != nil {
		return fmt.Errorf("stopping services: %w", err)
	}

	// Deregister from global agent registry.
	if err := registry.Deregister(af.Name); err != nil {
		p.Warn(&output.UserWarning{
			What:    fmt.Sprintf("Could not deregister agent from registry: %v", err),
			Assumed: "Services stopped but agent may still appear in `volra hub`",
		})
	}

	if removeVolumes {
		p.Warn(&output.UserWarning{
			What: "Volumes removed — monitoring data deleted",
		})
	}
	p.Result(fmt.Sprintf("Services stopped for %s", af.Name))

	return nil
}
