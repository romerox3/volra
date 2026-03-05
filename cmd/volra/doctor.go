package main

import (
	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites for Volra",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		r := docker.NewExecRunner()
		// Optionally load Agentfile for Level 2 checks (ignore errors — may not exist).
		af, _ := agentfile.Load("Agentfile")
		return doctor.Run(cmd.Context(), version, p, r, nil, af)
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
