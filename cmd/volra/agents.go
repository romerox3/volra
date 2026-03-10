package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/romerox3/volra/internal/controlplane"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var agentsLocalOnly bool

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List all agents across local and federated servers",
	Long:  "Show all agents in the mesh with their server, name, skills, and status.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runAgents(p)
	},
}

func init() {
	agentsCmd.Flags().BoolVar(&agentsLocalOnly, "local", false, "Show only local agents")
	rootCmd.AddCommand(agentsCmd)
}

func runAgents(p output.Presenter) error {
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	agents, err := store.ListAgents()
	if err != nil {
		return fmt.Errorf("listing agents: %w", err)
	}

	var peers []controlplane.FederationPeer
	if !agentsLocalOnly {
		peers, _ = store.ListPeers() // ignore error, just skip federation
	}

	client := controlplane.NewFederationClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	allAgents := client.AggregateAgents(ctx, agents, peers)

	if len(allAgents) == 0 {
		p.Result("No agents found")
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(allAgents, "", "  ")
		p.Result(string(data))
		return nil
	}

	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "SERVER\tAGENT\tFRAMEWORK\tPORT\tSTATUS")
	for _, a := range allAgents {
		framework := a.Framework
		if framework == "" {
			framework = "-"
		}
		port := "-"
		if a.Port > 0 {
			port = fmt.Sprintf("%d", a.Port)
		}
		status := a.Status
		if status == "" {
			status = "unknown"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			a.Server,
			a.Name,
			framework,
			port,
			status,
		)
	}
	w.Flush()

	p.Result(string(buf))

	// Summary.
	localCount := 0
	remoteCount := 0
	for _, a := range allAgents {
		if a.Server == "local" {
			localCount++
		} else {
			remoteCount++
		}
	}
	summary := fmt.Sprintf("%d local", localCount)
	if remoteCount > 0 {
		summary += fmt.Sprintf(", %d federated", remoteCount)
	}
	_ = strings.TrimSpace(summary)
	return nil
}
