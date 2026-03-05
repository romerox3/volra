// Package templates provides embedded quickstart templates for volra.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed basic rag conversational langgraph crewai openai-agents api-agent smolagents mcp-server discord-bot slack-bot web-chat
var content embed.FS

// Category groups templates in the interactive quickstart UI.
const (
	CategoryGettingStarted = "Getting Started"
	CategoryUseCase        = "Use Cases"
	CategoryFramework      = "Frameworks"
	CategoryPlatform       = "Platforms"
)

// Template describes an available quickstart template.
type Template struct {
	Name        string
	Description string
	Category    string
	Framework   string
	Services    []string
}

// templateMeta is a minimal struct for parsing Agentfile metadata.
type templateMeta struct {
	Framework string                 `yaml:"framework"`
	Services  map[string]interface{} `yaml:"services"`
}

// loadMeta reads the embedded Agentfile for a template and extracts framework + service keys.
func loadMeta(templateName string) (framework string, services []string) {
	data, err := fs.ReadFile(content, filepath.Join(templateName, "Agentfile"))
	if err != nil {
		return "generic", nil
	}

	var meta templateMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return "generic", nil
	}

	framework = meta.Framework
	if framework == "" {
		framework = "generic"
	}

	for k := range meta.Services {
		services = append(services, k)
	}

	return framework, services
}

// Available returns the list of available templates.
func Available() []Template {
	templates := []Template{
		// Getting Started
		{Name: "basic", Description: "Minimal FastAPI agent with health + ask endpoints", Category: CategoryGettingStarted},

		// Use Cases
		{Name: "rag", Description: "RAG agent with ChromaDB + Redis cache", Category: CategoryUseCase},
		{Name: "conversational", Description: "Conversational agent with LLM, Redis + PostgreSQL", Category: CategoryUseCase},
		{Name: "api-agent", Description: "Function-calling agent without any framework", Category: CategoryUseCase},
		{Name: "mcp-server", Description: "MCP-compatible tool server", Category: CategoryUseCase},

		// Frameworks
		{Name: "langgraph", Description: "LangGraph ReAct agent with tool-calling loop", Category: CategoryFramework},
		{Name: "crewai", Description: "CrewAI multi-agent research crew", Category: CategoryFramework},
		{Name: "openai-agents", Description: "OpenAI Agents SDK with tools and handoffs", Category: CategoryFramework},
		{Name: "smolagents", Description: "HuggingFace code agent with tool use", Category: CategoryFramework},

		// Platforms
		{Name: "discord-bot", Description: "AI-powered Discord bot with slash commands", Category: CategoryPlatform},
		{Name: "slack-bot", Description: "AI-powered Slack bot with event handling", Category: CategoryPlatform},
		{Name: "web-chat", Description: "Full-stack chat UI with WebSocket", Category: CategoryPlatform},
	}

	for i := range templates {
		templates[i].Framework, templates[i].Services = loadMeta(templates[i].Name)
	}

	return templates
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
