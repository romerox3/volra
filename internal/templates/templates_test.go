package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllTemplatesHaveRequiredFiles(t *testing.T) {
	required := []string{"Agentfile", "main.py", "requirements.txt", "README.md"}
	for _, tmpl := range Available() {
		t.Run(tmpl.Name, func(t *testing.T) {
			for _, file := range required {
				_, err := fs.ReadFile(content, filepath.Join(tmpl.Name, file))
				if err != nil {
					t.Errorf("template %q missing required file %q", tmpl.Name, file)
				}
			}
		})
	}
}

func TestScaffoldReplacesPlaceholders(t *testing.T) {
	for _, tmpl := range Available() {
		t.Run(tmpl.Name, func(t *testing.T) {
			dir := t.TempDir()
			projectName := "test-agent"

			if err := Scaffold(tmpl.Name, dir, projectName); err != nil {
				t.Fatalf("Scaffold(%q) failed: %v", tmpl.Name, err)
			}

			agentfilePath := filepath.Join(dir, "Agentfile")
			data, err := os.ReadFile(agentfilePath)
			if err != nil {
				t.Fatalf("reading Agentfile: %v", err)
			}

			content := string(data)
			if strings.Contains(content, "{{.Name}}") {
				t.Error("Agentfile still contains unreplaced {{.Name}} placeholder")
			}
			if !strings.Contains(content, projectName) {
				t.Errorf("Agentfile does not contain project name %q", projectName)
			}
		})
	}
}

func TestScaffoldCreatesAllFiles(t *testing.T) {
	for _, tmpl := range Available() {
		t.Run(tmpl.Name, func(t *testing.T) {
			dir := t.TempDir()

			if err := Scaffold(tmpl.Name, dir, "my-project"); err != nil {
				t.Fatalf("Scaffold(%q) failed: %v", tmpl.Name, err)
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("reading dir: %v", err)
			}
			if len(entries) < 4 {
				t.Errorf("expected at least 4 files, got %d", len(entries))
			}
		})
	}
}

func TestUnknownTemplateReturnsError(t *testing.T) {
	err := Scaffold("nonexistent-template", t.TempDir(), "test")
	if err == nil {
		t.Error("expected error for unknown template, got nil")
	}
}
