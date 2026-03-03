package main

import (
	"github.com/antonioromero/volra/internal/docker"
	"github.com/antonioromero/volra/internal/doctor"
	"github.com/antonioromero/volra/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites for Volra",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := output.NewPresenter(output.DetectMode())
		r := docker.NewExecRunner()
		return doctor.Run(cmd.Context(), version, p, r, nil)
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
