package setup

import (
	"regexp"

	"github.com/antonioromero/volra/internal/output"
)

const defaultHealthPath = "/health"

// healthPatterns matches common Python route decorators with health-related paths.
var healthPatterns = []*regexp.Regexp{
	regexp.MustCompile(`@app\.(get|route)\(\s*["'](/[^"']*health[^"']*)["']`),
	regexp.MustCompile(`@router\.(get|route)\(\s*["'](/[^"']*health[^"']*)["']`),
}

// detectHealthPath extracts the health check endpoint from entry point code.
func detectHealthPath(entryCode string) (string, *output.UserWarning) {
	for _, pat := range healthPatterns {
		if m := pat.FindStringSubmatch(entryCode); len(m) > 2 {
			return m[2], nil
		}
	}
	return defaultHealthPath, &output.UserWarning{
		What:     "Health endpoint not detected in entry point code",
		Assumed:  "/health",
		Override: "Edit Agentfile and set 'health_path' to your health check endpoint",
	}
}
