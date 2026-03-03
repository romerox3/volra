package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// GenerateDockerfile renders the Dockerfile template and writes it to the output directory.
func GenerateDockerfile(tc *TemplateContext, dir string) error {
	tmplData, err := templateFS.ReadFile("templates/Dockerfile.tmpl")
	if err != nil {
		return fmt.Errorf("reading Dockerfile template: %w", err)
	}

	tmpl, err := template.New("Dockerfile").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parsing Dockerfile template: %w", err)
	}

	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "Dockerfile")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating Dockerfile: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, tc); err != nil {
		return fmt.Errorf("rendering Dockerfile: %w", err)
	}

	return nil
}

// RenderDockerfile renders the Dockerfile template to a string (for testing).
func RenderDockerfile(tc *TemplateContext) (string, error) {
	tmplData, err := templateFS.ReadFile("templates/Dockerfile.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading Dockerfile template: %w", err)
	}

	tmpl, err := template.New("Dockerfile").Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("parsing Dockerfile template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, tc); err != nil {
		return "", fmt.Errorf("rendering Dockerfile: %w", err)
	}

	return buf.String(), nil
}
