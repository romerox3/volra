package main

import (
	"os"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for editor integration",
	Long:  "Start a Model Context Protocol server on stdio. Used by editors like Cursor, VS Code, and Claude Code to integrate with Volra.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return mcp.Serve(cmd.Context(), os.Stdin, os.Stdout, version)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
