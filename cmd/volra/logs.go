package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "Stream logs from deployed agent",
	Long:  "Show logs from the deployed agent or a specific service. Defaults to the agent container.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

var (
	logsFollow bool
	logsLines  int
)

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 100, "Number of lines to show from the end")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	af, err := agentfile.Load("Agentfile")
	if err != nil {
		return fmt.Errorf("no Agentfile found — are you in an agent project directory?")
	}

	composePath := filepath.Join(".volra", "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("no deployment found — run 'volra deploy' first")
	}

	// Build docker compose logs command.
	composeArgs := []string{"compose", "-f", composePath, "logs"}
	composeArgs = append(composeArgs, "--tail", strconv.Itoa(logsLines))

	if logsFollow {
		composeArgs = append(composeArgs, "--follow")
	}

	// Determine target service.
	if len(args) > 0 {
		// User specified a service name — prefix with agent name.
		composeArgs = append(composeArgs, af.Name+"-"+args[0])
	} else {
		// Default: the agent container.
		composeArgs = append(composeArgs, af.Name)
	}

	// Stream directly to stdout/stderr (don't capture output).
	c := exec.CommandContext(cmd.Context(), "docker", composeArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
