package main

import (
	"github.com/romerox3/volra/internal/setup"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a project for Volra deployment",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := newPresenter()
		defer flushPresenter(p)
		return setup.Run(cmd.Context(), dir, initForce, p)
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing Agentfile")
	rootCmd.AddCommand(initCmd)
}
