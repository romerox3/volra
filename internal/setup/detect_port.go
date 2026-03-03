package setup

import (
	"regexp"
	"strconv"

	"github.com/antonioromero/volra/internal/output"
)

const defaultPort = 8000

// portPatterns matches common Python server port declarations.
var portPatterns = []*regexp.Regexp{
	regexp.MustCompile(`uvicorn\.run\([^)]*port\s*=\s*(\d+)`),
	regexp.MustCompile(`\.run\([^)]*port\s*=\s*(\d+)`),
	regexp.MustCompile(`--port\s+(\d+)`),
}

// detectPort extracts the port from entry point code using regex patterns.
func detectPort(entryCode string) (int, *output.UserWarning) {
	for _, pat := range portPatterns {
		if m := pat.FindStringSubmatch(entryCode); len(m) > 1 {
			port, err := strconv.Atoi(m[1])
			if err == nil && port >= 1 && port <= 65535 {
				return port, nil
			}
		}
	}
	return defaultPort, &output.UserWarning{
		What:     "Port not detected in entry point code",
		Assumed:  "8000",
		Override: "Edit Agentfile and set 'port' to your application's port",
	}
}
