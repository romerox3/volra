package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// GeneratePrometheus renders the prometheus.yml template and writes it to the output directory.
func GeneratePrometheus(tc *TemplateContext, dir string) error {
	content, err := RenderPrometheus(tc)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "prometheus.yml")
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// RenderPrometheus renders the prometheus.yml template to a string (for testing).
func RenderPrometheus(tc *TemplateContext) (string, error) {
	tmplData, err := templateFS.ReadFile("templates/prometheus.yml.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading prometheus template: %w", err)
	}

	tmpl, err := template.New("prometheus").Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("parsing prometheus template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, tc); err != nil {
		return "", fmt.Errorf("rendering prometheus: %w", err)
	}

	return buf.String(), nil
}

// CopyAlertRules copies the static alert_rules.yml to the output directory.
func CopyAlertRules(dir string) error {
	data, err := staticFS.ReadFile("static/alert_rules.yml")
	if err != nil {
		return fmt.Errorf("reading alert rules: %w", err)
	}

	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "alert_rules.yml")
	return os.WriteFile(outputPath, data, 0644)
}
