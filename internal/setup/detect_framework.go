package setup

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
)

// detectFramework scans dependency files for known framework packages.
// Detection priority: LangGraph > CrewAI > Generic.
func detectFramework(dir string) agentfile.Framework {
	depFiles := []struct {
		path string
		scan func(string, string) bool
	}{
		{filepath.Join(dir, "requirements.txt"), scanRequirementsTxtFor},
		{filepath.Join(dir, "pyproject.toml"), scanContentFor},
		{filepath.Join(dir, "Pipfile"), scanContentFor},
	}

	// Check LangGraph first (highest priority).
	for _, df := range depFiles {
		if df.scan(df.path, "langgraph") {
			return agentfile.FrameworkLangGraph
		}
	}

	// Check CrewAI second.
	for _, df := range depFiles {
		if df.scan(df.path, "crewai") {
			return agentfile.FrameworkCrewAI
		}
	}

	return agentfile.FrameworkGeneric
}

// scanRequirementsTxtFor checks if a package appears in requirements.txt by exact name match.
func scanRequirementsTxtFor(path, pkg string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := extractPackageName(line)
		if strings.EqualFold(name, pkg) {
			return true
		}
	}
	return false
}

// scanContentFor checks if a package name appears in a file's content (pyproject.toml, Pipfile).
func scanContentFor(path, pkg string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), strings.ToLower(pkg))
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
