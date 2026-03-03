package setup

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/antonioromero/volra/internal/agentfile"
)

// detectFramework scans dependency files for known framework packages.
func detectFramework(dir string) agentfile.Framework {
	if scanRequirementsTxt(filepath.Join(dir, "requirements.txt")) {
		return agentfile.FrameworkLangGraph
	}
	if scanPyprojectToml(filepath.Join(dir, "pyproject.toml")) {
		return agentfile.FrameworkLangGraph
	}
	return agentfile.FrameworkGeneric
}

// scanRequirementsTxt checks if langgraph appears in requirements.txt.
func scanRequirementsTxt(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Extract package name before version specifiers.
		pkg := extractPackageName(line)
		if strings.EqualFold(pkg, "langgraph") {
			return true
		}
	}
	return false
}

// scanPyprojectToml checks if langgraph appears in pyproject.toml dependencies.
func scanPyprojectToml(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Simple content scan — look for langgraph in dependency arrays.
	// Works for both [project].dependencies and [project.optional-dependencies].
	return strings.Contains(strings.ToLower(string(data)), "langgraph")
}

// extractPackageName strips version specifiers from a requirements.txt line.
func extractPackageName(line string) string {
	for i, c := range line {
		if c == '=' || c == '>' || c == '<' || c == '!' || c == '~' || c == '[' {
			return strings.TrimSpace(line[:i])
		}
	}
	return strings.TrimSpace(line)
}
