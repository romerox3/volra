package setup

import (
	"os"
	"path/filepath"

	"github.com/antonioromero/volra/internal/output"
)

// entryPointCandidates is the priority-ordered list of common Python entry points.
var entryPointCandidates = []string{"main.py", "app.py", "server.py"}

const defaultEntryPoint = "main.py"

// detectEntryPoint finds the first matching entry point file in priority order.
func detectEntryPoint(dir string) (string, *output.UserWarning) {
	for _, candidate := range entryPointCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			return candidate, nil
		}
	}
	return defaultEntryPoint, &output.UserWarning{
		What:     "No entry point detected (main.py, app.py, or server.py)",
		Assumed:  defaultEntryPoint,
		Override: "Edit Agentfile and set the correct entry point filename",
	}
}
