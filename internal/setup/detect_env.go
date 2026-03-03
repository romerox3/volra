package setup

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// envPatterns matches common Python environment variable access patterns.
var envPatterns = []*regexp.Regexp{
	regexp.MustCompile(`os\.environ\[["']([A-Z_][A-Z0-9_]*)["']\]`),
	regexp.MustCompile(`os\.environ\.get\(["']([A-Z_][A-Z0-9_]*)["']`),
	regexp.MustCompile(`os\.getenv\(["']([A-Z_][A-Z0-9_]*)["']`),
}

// detectEnvVars scans all Python files in a directory for environment variable references.
func detectEnvVars(dir string) []string {
	seen := make(map[string]bool)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		content := readFileContents(dir, e.Name())
		for _, pat := range envPatterns {
			matches := pat.FindAllStringSubmatch(content, -1)
			for _, m := range matches {
				if len(m) > 1 {
					seen[m[1]] = true
				}
			}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	vars := make([]string, 0, len(seen))
	for v := range seen {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	return vars
}

// readFileContents reads a file's contents, returning empty string on error.
func readFileContents(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return string(data)
}
