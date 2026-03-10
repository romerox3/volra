package deploy

import (
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
)

// DeployHealthcheck is the deploy-time representation of a healthcheck.
type DeployHealthcheck struct {
	Test        []string
	Interval    string
	Timeout     string
	Retries     int
	StartPeriod string
}

// DeployResources is the deploy-time representation of resource limits.
type DeployResources struct {
	MemLimit string
	CPUs     string
}

// knownHealthchecks maps image prefixes to default healthcheck configurations.
var knownHealthchecks = map[string]DeployHealthcheck{
	"postgres": {
		Test: []string{"CMD-SHELL", "pg_isready -U postgres || exit 1"},
		Interval: "5s", Timeout: "3s", Retries: 5, StartPeriod: "10s",
	},
	"redis": {
		Test: []string{"CMD-SHELL", "redis-cli ping | grep -q PONG"},
		Interval: "5s", Timeout: "3s", Retries: 5, StartPeriod: "5s",
	},
	// chromadb/chroma:latest is a Rust binary without shell — auto-healthcheck not possible.
	// Users can add an explicit healthcheck in the Agentfile if needed.
}

// knownResources maps image prefixes to default resource limits.
var knownResources = map[string]DeployResources{
	"redis":           {MemLimit: "256m", CPUs: "0.25"},
	"postgres":        {MemLimit: "512m", CPUs: "0.5"},
	"chromadb/chroma": {MemLimit: "1g", CPUs: "1.0"},
}

// imageMatchesPrefix checks if an image string matches a known prefix.
// Matches: "prefix:tag", "prefix/sub", or exact "prefix".
func imageMatchesPrefix(image, prefix string) bool {
	if image == prefix {
		return true
	}
	if strings.HasPrefix(image, prefix+":") {
		return true
	}
	// For images like "chromadb/chroma:0.4.24"
	if strings.HasPrefix(image, prefix+"/") {
		return true
	}
	return false
}

// ensureCMDPrefix adds "CMD" prefix to healthcheck test arrays if not already present.
// Docker Compose requires test arrays to start with "CMD", "CMD-SHELL", or "NONE".
func ensureCMDPrefix(test []string) []string {
	if len(test) == 0 {
		return test
	}
	first := strings.ToUpper(test[0])
	if first == "CMD" || first == "CMD-SHELL" || first == "NONE" {
		return test
	}
	return append([]string{"CMD"}, test...)
}

// resolveHealthcheck returns a healthcheck for the given image.
// Explicit config takes priority; otherwise uses known defaults.
func resolveHealthcheck(image string, explicit *agentfile.HealthcheckConfig) *DeployHealthcheck {
	if explicit != nil {
		return &DeployHealthcheck{
			Test:        ensureCMDPrefix(explicit.Test),
			Interval:    explicit.Interval,
			Timeout:     explicit.Timeout,
			Retries:     explicit.Retries,
			StartPeriod: explicit.StartPeriod,
		}
	}
	for prefix, hc := range knownHealthchecks {
		if imageMatchesPrefix(image, prefix) {
			copy := hc
			return &copy
		}
	}
	return nil
}

// resolveResources returns resource limits for the given image.
// Explicit config takes priority; otherwise uses known defaults.
func resolveResources(image string, explicit *agentfile.ResourceConfig) *DeployResources {
	if explicit != nil {
		return &DeployResources{
			MemLimit: explicit.MemLimit,
			CPUs:     explicit.CPUs,
		}
	}
	for prefix, res := range knownResources {
		if imageMatchesPrefix(image, prefix) {
			copy := res
			return &copy
		}
	}
	return nil
}
