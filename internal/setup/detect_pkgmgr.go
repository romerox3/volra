package setup

import (
	"os"
	"path/filepath"

	"github.com/romerox3/volra/internal/agentfile"
)

// detectPackageManager detects the Python package manager by checking for lockfiles.
// Priority: uv.lock > poetry.lock > Pipfile.lock > pip (default).
func detectPackageManager(dir string) agentfile.PackageManager {
	if fileExists(dir, "uv.lock") {
		return agentfile.PackageManagerUV
	}
	if fileExists(dir, "poetry.lock") {
		return agentfile.PackageManagerPoetry
	}
	if fileExists(dir, "Pipfile.lock") {
		return agentfile.PackageManagerPipenv
	}
	return agentfile.PackageManagerPip
}

// fileExists checks if a file exists in the given directory.
func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
