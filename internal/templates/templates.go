// Package templates provides embedded quickstart templates for volra.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed basic rag conversational
var content embed.FS

// Template describes an available quickstart template.
type Template struct {
	Name        string
	Description string
}

// Available returns the list of available templates.
func Available() []Template {
	return []Template{
		{Name: "basic", Description: "Minimal FastAPI agent with health + ask endpoints"},
		{Name: "rag", Description: "RAG agent with Redis cache"},
		{Name: "conversational", Description: "Conversational agent with Redis + PostgreSQL"},
	}
}

// Scaffold copies a template to the target directory, replacing {{.Name}} placeholders.
func Scaffold(templateName, targetDir, projectName string) error {
	// Verify template exists.
	entries, err := fs.ReadDir(content, templateName)
	if err != nil {
		return fmt.Errorf("unknown template: %s", templateName)
	}

	// Create target directory.
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Copy each file with placeholder replacement.
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := fs.ReadFile(content, filepath.Join(templateName, entry.Name()))
		if err != nil {
			return fmt.Errorf("reading template file %s: %w", entry.Name(), err)
		}

		// Replace placeholders.
		output := strings.ReplaceAll(string(data), "{{.Name}}", projectName)

		outPath := filepath.Join(targetDir, entry.Name())
		if err := os.WriteFile(outPath, []byte(output), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}

	return nil
}
