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

func TestTemplateMetadata(t *testing.T) {
	for _, tmpl := range Available() {
		t.Run(tmpl.Name, func(t *testing.T) {
			if tmpl.Framework == "" {
				t.Error("Framework must not be empty")
			}
		})
	}
}

func TestTemplateServicesCount(t *testing.T) {
	expected := map[string]int{
		"basic":              0,
		"rag":                2, // cache + vectordb
		"conversational":     2, // redis + postgres
		"api-agent":          0,
		"mcp-server":         0,
		"langgraph":          0,
		"crewai":             0,
		"openai-agents":      0,
		"smolagents":         0,
		"discord-bot":        1, // cache
		"slack-bot":          0,
		"web-chat":           0,
		"custom-agent":       0,
		"fastapi-bot":        0,
		"langchain-chatbot":  0,
		"langchain-agent":    0,
		"langchain-rag":      1, // chromadb
		"openai-assistant":   0,
		"openai-swarm":       0,
		"crewai-team":        0,
		"crewai-researcher":  0,
		"autogen-duo":        0,
		"autogen-group":      0,
		"pgvector-rag":       1, // postgres
	}

	for _, tmpl := range Available() {
		t.Run(tmpl.Name, func(t *testing.T) {
			want, ok := expected[tmpl.Name]
			if !ok {
				t.Skipf("no expected count for %s", tmpl.Name)
			}
			if got := len(tmpl.Services); got != want {
				t.Errorf("services count = %d, want %d (services: %v)", got, want, tmpl.Services)
			}
		})
	}
}

func TestTemplateLangGraphFramework(t *testing.T) {
	for _, tmpl := range Available() {
		if tmpl.Name == "langgraph" {
			if tmpl.Framework != "langgraph" {
				t.Errorf("langgraph template framework = %q, want %q", tmpl.Framework, "langgraph")
			}
			return
		}
	}
	t.Error("langgraph template not found")
}
