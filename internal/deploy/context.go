package deploy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/antonioromero/volra/internal/agentfile"
)

// TemplateContext extends Agentfile with deploy-time detected metadata.
type TemplateContext struct {
	agentfile.Agentfile
	PythonVersion   string
	EntryPoint      string
	HasRequirements bool
}

// JobHealth returns the Prometheus job name for health endpoint scraping.
func (tc *TemplateContext) JobHealth() string { return JobHealth }

// JobMetrics returns the Prometheus job name for metrics scraping.
func (tc *TemplateContext) JobMetrics() string { return JobMetrics }

// entryPointCandidates in priority order for deploy-time detection.
var deployEntryPointCandidates = []string{"main.py", "app.py", "server.py"}

// BuildContext creates a TemplateContext by combining Agentfile with deploy-time detection.
func BuildContext(af *agentfile.Agentfile, dir string) *TemplateContext {
	tc := &TemplateContext{
		Agentfile:     *af,
		PythonVersion: detectPythonVersion(dir),
		EntryPoint:    detectDeployEntryPoint(dir),
	}
	_, err := os.Stat(filepath.Join(dir, "requirements.txt"))
	tc.HasRequirements = err == nil
	return tc
}

// detectPythonVersion tries to extract Python version from pyproject.toml, falls back to "3.11".
func detectPythonVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return "3.11"
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "requires-python") {
			// Extract version like >=3.12 → 3.12
			for i, c := range trimmed {
				if c >= '0' && c <= '9' {
					ver := strings.TrimSpace(trimmed[i:])
					ver = strings.Trim(ver, "\"'")
					// Take major.minor only
					parts := strings.SplitN(ver, ".", 3)
					if len(parts) >= 2 {
						return parts[0] + "." + parts[1]
					}
					return ver
				}
				_ = i
			}
		}
	}
	return "3.11"
}

// detectDeployEntryPoint finds the entry point at deploy time.
func detectDeployEntryPoint(dir string) string {
	for _, candidate := range deployEntryPointCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			return candidate
		}
	}
	return "main.py"
}
