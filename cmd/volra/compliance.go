package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/compliance"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "EU AI Act compliance documentation",
	Long:  "Generate and manage EU AI Act compliance documentation for deployed agents.",
}

var complianceGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate compliance documentation",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runComplianceGenerate(p)
	},
}

func init() {
	complianceCmd.AddCommand(complianceGenerateCmd)
	rootCmd.AddCommand(complianceCmd)
}

func runComplianceGenerate(p output.Presenter) error {
	af, err := agentfile.Load("Agentfile")
	if err != nil {
		return &output.UserError{
			Code: output.CodeNoAgentfileForCompliance,
			What: "No Agentfile found in current directory",
			Fix:  "Run 'volra init .' to create an Agentfile first",
		}
	}

	doc, err := compliance.Generate(af)
	if err != nil {
		return fmt.Errorf("generating compliance doc: %w", err)
	}

	outputDir := ".volra"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "compliance.md")
	if err := os.WriteFile(outputPath, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("writing compliance doc: %w", err)
	}

	p.Result(fmt.Sprintf("Compliance doc generated: %s", outputPath))
	return nil
}
