// Package registry manages the global agent registry at ~/.volra/agents.json.
// Deploy registers agents, down deregisters them, hub reads the list.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AgentEntry represents a registered agent in the global registry.
type AgentEntry struct {
	Name           string    `json:"name"`
	ProjectDir     string    `json:"project_dir"`
	PrometheusPort int       `json:"prometheus_port"`
	AgentPort      int       `json:"agent_port"`
	RegisteredAt   time.Time `json:"registered_at"`
	Status         string    `json:"status"`
}

// Registry holds the list of registered agents.
type Registry struct {
	Agents []AgentEntry `json:"agents"`
}

// PathFunc returns the registry file path. Replaceable for testing.
var PathFunc = defaultPath

func defaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".volra", "agents.json"), nil
}

// Register adds or updates an agent entry in the global registry.
func Register(name, projectDir string, prometheusPort, agentPort int) error {
	reg, err := load()
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolving project path: %w", err)
	}

	entry := AgentEntry{
		Name:           name,
		ProjectDir:     absDir,
		PrometheusPort: prometheusPort,
		AgentPort:      agentPort,
		RegisteredAt:   time.Now().UTC(),
		Status:         "deployed",
	}

	// Update existing entry if same name + same dir, otherwise append.
	updated := false
	for i, a := range reg.Agents {
		if a.Name == name && a.ProjectDir == absDir {
			reg.Agents[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		reg.Agents = append(reg.Agents, entry)
	}

	return save(reg)
}

// Deregister removes an agent by name from the global registry.
func Deregister(name string) error {
	reg, err := load()
	if err != nil {
		return err
	}

	filtered := make([]AgentEntry, 0, len(reg.Agents))
	for _, a := range reg.Agents {
		if a.Name != name {
			filtered = append(filtered, a)
		}
	}
	reg.Agents = filtered

	return save(reg)
}

// List returns all registered agents.
func List() ([]AgentEntry, error) {
	reg, err := load()
	if err != nil {
		return nil, err
	}
	return reg.Agents, nil
}

func load() (*Registry, error) {
	path, err := PathFunc()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	return &reg, nil
}

func save(reg *Registry) error {
	path, err := PathFunc()
	if err != nil {
		return err
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}

	// Atomic write: temp file + rename.
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("writing registry temp file: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("renaming registry temp file: %w", err)
	}

	return nil
}
