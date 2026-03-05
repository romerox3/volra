package deploy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
)

// GenerateEnvFiles reads the source .env file and generates per-service env files
// in the .volra output directory, each containing only the variables declared by that service.
func GenerateEnvFiles(af *agentfile.Agentfile, dir string) error {
	// Parse source .env file
	envMap, err := parseEnvFile(filepath.Join(dir, ".env"))
	if err != nil {
		return fmt.Errorf("reading .env: %w", err)
	}

	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate agent.env
	if len(af.Env) > 0 {
		if err := writeEnvFile(filepath.Join(outputDir, "agent.env"), af.Env, envMap); err != nil {
			return fmt.Errorf("writing agent.env: %w", err)
		}
	}

	// Generate per-service env files
	for name, svc := range af.Services {
		if len(svc.Env) > 0 {
			filename := fmt.Sprintf("%s-%s.env", af.Name, name)
			if err := writeEnvFile(filepath.Join(outputDir, filename), svc.Env, envMap); err != nil {
				return fmt.Errorf("writing %s: %w", filename, err)
			}
		}
	}

	return nil
}

// parseEnvFile reads a .env file and returns a map of KEY=VALUE pairs.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.IndexByte(line, '='); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			result[key] = value
		}
	}
	return result, scanner.Err()
}

// writeEnvFile writes only the specified keys from envMap to the output file.
func writeEnvFile(path string, keys []string, envMap map[string]string) error {
	var lines []string
	for _, key := range keys {
		if val, ok := envMap[key]; ok {
			lines = append(lines, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
