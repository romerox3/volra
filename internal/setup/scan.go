package setup

import (
	"github.com/antonioromero/volra/internal/agentfile"
	"github.com/antonioromero/volra/internal/output"
)

// ScanResult holds all auto-detected project properties.
type ScanResult struct {
	Framework  agentfile.Framework
	EntryPoint string
	Port       int
	HealthPath string
	EnvVars    []string
	Warnings   []*output.UserWarning
}

// ScanProject scans a Python project directory and detects configuration values.
func ScanProject(dir string) *ScanResult {
	result := &ScanResult{}

	result.Framework = detectFramework(dir)

	entry, entryWarn := detectEntryPoint(dir)
	result.EntryPoint = entry
	if entryWarn != nil {
		result.Warnings = append(result.Warnings, entryWarn)
	}

	entryCode := readFileContents(dir, entry)

	port, portWarn := detectPort(entryCode)
	result.Port = port
	if portWarn != nil {
		result.Warnings = append(result.Warnings, portWarn)
	}

	health, healthWarn := detectHealthPath(entryCode)
	result.HealthPath = health
	if healthWarn != nil {
		result.Warnings = append(result.Warnings, healthWarn)
	}

	result.EnvVars = detectEnvVars(dir)

	return result
}
