package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:     "volra",
	Short:   "Deploy AI agents to production with monitoring",
	Version: version,
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}
