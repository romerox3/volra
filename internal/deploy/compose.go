package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// GenerateCompose renders the docker-compose.yml template and writes it to the output directory.
func GenerateCompose(tc *TemplateContext, dir string) error {
	content, err := RenderCompose(tc)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "docker-compose.yml")
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// RenderCompose renders the docker-compose.yml template to a string (for testing).
func RenderCompose(tc *TemplateContext) (string, error) {
	tmplData, err := templateFS.ReadFile("templates/docker-compose.yml.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading compose template: %w", err)
	}

	funcMap := template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"gt": func(a, b int) bool { return a > b },
	}

	tmpl, err := template.New("compose").Funcs(funcMap).Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("parsing compose template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, tc); err != nil {
		return "", fmt.Errorf("rendering compose: %w", err)
	}

	return buf.String(), nil
}
