// Package status implements the volra status command logic.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/antonioromero/volra/internal/agentfile"
	"github.com/antonioromero/volra/internal/docker"
	"github.com/antonioromero/volra/internal/output"
)

const outputDir = ".volra"

// ServiceState represents the running state of a Docker service.
type ServiceState struct {
	Name  string
	State string // running, exited, etc.
	Port  string
}

// composeService is used to parse docker compose ps JSON output.
type composeService struct {
	Name  string `json:"Name"`
	State string `json:"State"`
}

// Run executes the status command pipeline:
// Load Agentfile → Check Docker → Query containers → Probe health → Report.
func Run(ctx context.Context, dir string, p output.Presenter, dr docker.DockerRunner) error {
	// 1. Load Agentfile
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return &output.UserError{
			Code: output.CodeNoDeployment,
			What: "No deployment found — Agentfile not found",
			Fix:  "Run: volra init && volra deploy",
		}
	}

	// 2. Check Docker is running
	_, err = dr.Run(ctx, "info")
	if err != nil {
		return &output.UserError{
			Code: output.CodeStatusDockerNotRunning,
			What: "Docker is not running",
			Fix:  "Start Docker Desktop or the Docker daemon",
		}
	}

	// 3. Query container states
	composePath := filepath.Join(dir, outputDir, "docker-compose.yml")
	services, err := queryContainers(ctx, dr, composePath, af.Name)
	if err != nil {
		return &output.UserError{
			Code: output.CodeNoDeployment,
			What: "No deployment found — docker compose project not running",
			Fix:  "Run: volra deploy",
		}
	}

	// 4. Detect daemon restart (all containers stopped/exited)
	if allStopped(services) {
		p.Warn(&output.UserWarning{
			What:     "All containers are stopped — Docker may have restarted",
			Override: "Run: volra deploy",
		})
	}

	// 5. Probe agent health
	healthy := probeHealth(ctx, af.Port, af.HealthPath)

	// 6. Report
	reportServices(p, services, af, healthy)

	return nil
}

// queryContainers gets container states via docker compose ps.
func queryContainers(ctx context.Context, dr docker.DockerRunner, composePath string, name string) ([]ServiceState, error) {
	out, err := dr.Run(ctx, "compose", "-f", composePath, "-p", name, "ps", "--format", "json")
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, fmt.Errorf("no containers found")
	}

	var services []ServiceState

	// docker compose ps --format json outputs one JSON object per line
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var svc composeService
		if err := json.Unmarshal([]byte(line), &svc); err != nil {
			continue
		}
		services = append(services, ServiceState{
			Name:  svc.Name,
			State: svc.State,
		})
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no containers found")
	}

	return services, nil
}

// allStopped returns true if all containers are stopped/exited.
func allStopped(services []ServiceState) bool {
	if len(services) == 0 {
		return false
	}
	for _, s := range services {
		state := strings.ToLower(s.State)
		if state != "exited" && state != "dead" && state != "created" {
			return false
		}
	}
	return true
}

// probeHealth makes a single HTTP request to the agent's health endpoint.
func probeHealth(ctx context.Context, port int, healthPath string) bool {
	url := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// reportServices outputs status for each service.
func reportServices(p output.Presenter, services []ServiceState, af *agentfile.Agentfile, healthy bool) {
	// Agent status
	agentState := findState(services, "agent")
	healthLabel := "unhealthy"
	if healthy {
		healthLabel = "healthy"
	}
	p.Result(fmt.Sprintf("Agent:      %s (%s) — port %d", agentState, healthLabel, af.Port))

	if !healthy && agentState == "running" {
		p.Warn(&output.UserWarning{
			What:     "Agent is running but health check failed",
			Override: fmt.Sprintf("Check logs: docker logs %s-agent", af.Name),
		})
	}

	// Prometheus status
	promState := findState(services, "prometheus")
	p.Result(fmt.Sprintf("Prometheus: %s — port 9090", promState))

	// Grafana status
	grafState := findState(services, "grafana")
	p.Result(fmt.Sprintf("Grafana:    %s — port 3001", grafState))
}

// findState finds the state of a service by partial name match.
func findState(services []ServiceState, suffix string) string {
	for _, s := range services {
		if strings.HasSuffix(s.Name, "-"+suffix) || strings.Contains(s.Name, suffix) {
			return s.State
		}
	}
	return "not found"
}
