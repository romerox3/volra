// Package hub manages the unified multi-agent dashboard (Prometheus federation + Grafana).
package hub

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

const (
	hubDirName         = "hub"
	defaultPromPort    = 9099
	defaultGrafanaPort = 3000
	projectName        = "volra-hub"
)

// composeContext holds template data for docker-compose.yml.
type composeContext struct {
	PrometheusPort int
	GrafanaPort    int
	ExtraHosts     bool
}

// promContext holds template data for prometheus.yml.
type promContext struct {
	Agents []registry.AgentEntry
}

// HubDirFunc returns the hub directory path. Replaceable for testing.
var HubDirFunc = defaultHubDir

func defaultHubDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".volra", hubDirName), nil
}

// Start generates federation config and starts hub Prometheus + Grafana.
func Start(ctx context.Context, p output.Presenter, dr docker.DockerRunner) error {
	agents, err := registry.List()
	if err != nil {
		return fmt.Errorf("reading agent registry: %w", err)
	}
	if len(agents) == 0 {
		return &output.UserError{
			Code: output.CodeNoAgentsRegistered,
			What: "No agents registered",
			Fix:  "Deploy at least one agent with `volra deploy` first",
		}
	}

	hubDir, err := HubDirFunc()
	if err != nil {
		return err
	}

	// Check if hub is already running.
	composePath := filepath.Join(hubDir, "docker-compose.yml")
	if _, statErr := os.Stat(composePath); statErr == nil {
		// Check if containers are running (ps -q outputs container IDs if running).
		psOut, runErr := dr.Run(ctx, "compose", "-f", composePath, "-p", projectName, "ps", "--quiet")
		if runErr == nil && strings.TrimSpace(psOut) != "" {
			return &output.UserError{
				Code: output.CodeHubAlreadyRunning,
				What: "Hub is already running",
				Fix:  "Run `volra hub stop` first, then `volra hub start`",
			}
		}
	}

	p.Progress(fmt.Sprintf("Generating hub config for %d agents...", len(agents)))

	if err := generateHubArtifacts(hubDir, agents); err != nil {
		return fmt.Errorf("generating hub artifacts: %w", err)
	}

	p.Progress("Starting hub services...")
	_, err = dr.Run(ctx, "compose", "-f", composePath, "-p", projectName, "up", "-d")
	if err != nil {
		return fmt.Errorf("starting hub services: %w", err)
	}

	p.Result(fmt.Sprintf("Hub Grafana:    http://localhost:%d", defaultGrafanaPort))
	p.Result(fmt.Sprintf("Hub Prometheus: http://localhost:%d", defaultPromPort))
	p.Result(fmt.Sprintf("Agents:         %d registered", len(agents)))

	return nil
}

// Stop tears down the hub services.
func Stop(ctx context.Context, p output.Presenter, dr docker.DockerRunner) error {
	hubDir, err := HubDirFunc()
	if err != nil {
		return err
	}

	composePath := filepath.Join(hubDir, "docker-compose.yml")
	if _, statErr := os.Stat(composePath); os.IsNotExist(statErr) {
		p.Result("Hub is not running")
		return nil
	}

	_, err = dr.Run(ctx, "compose", "-f", composePath, "-p", projectName, "down")
	if err != nil {
		return fmt.Errorf("stopping hub services: %w", err)
	}

	p.Result("Hub services stopped")
	return nil
}

// Status reports hub state and registered agents.
func Status(ctx context.Context, p output.Presenter, dr docker.DockerRunner) error {
	agents, err := registry.List()
	if err != nil {
		return fmt.Errorf("reading agent registry: %w", err)
	}

	if len(agents) == 0 {
		p.Result("No agents registered")
		return nil
	}

	hubDir, err := HubDirFunc()
	if err != nil {
		return err
	}

	composePath := filepath.Join(hubDir, "docker-compose.yml")
	hubRunning := false
	if _, statErr := os.Stat(composePath); statErr == nil {
		psOut, runErr := dr.Run(ctx, "compose", "-f", composePath, "-p", projectName, "ps", "--quiet")
		hubRunning = runErr == nil && strings.TrimSpace(psOut) != ""
	}

	if hubRunning {
		p.Progress("Hub: running")
	} else {
		p.Progress("Hub: stopped")
	}

	for _, a := range agents {
		p.Result(fmt.Sprintf("  %-20s agent=:%d  prometheus=:%d  %s", a.Name, a.AgentPort, a.PrometheusPort, a.Status))
	}

	return nil
}

func generateHubArtifacts(hubDir string, agents []registry.AgentEntry) error {
	// Create directory structure.
	dirs := []string{
		hubDir,
		filepath.Join(hubDir, "grafana", "provisioning", "datasources"),
		filepath.Join(hubDir, "grafana", "provisioning", "dashboards"),
		filepath.Join(hubDir, "grafana", "dashboards"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Generate prometheus.yml.
	if err := renderTemplate("templates/prometheus.yml.tmpl", filepath.Join(hubDir, "prometheus.yml"), promContext{Agents: agents}); err != nil {
		return err
	}

	// Generate docker-compose.yml.
	cc := composeContext{
		PrometheusPort: defaultPromPort,
		GrafanaPort:    defaultGrafanaPort,
		ExtraHosts:     true,
	}
	if err := renderTemplate("templates/docker-compose.yml.tmpl", filepath.Join(hubDir, "docker-compose.yml"), cc); err != nil {
		return err
	}

	// Copy Grafana provisioning files.
	if err := writeStaticFile("static/datasource.yml", filepath.Join(hubDir, "grafana", "provisioning", "datasources", "datasource.yml")); err != nil {
		return err
	}
	if err := writeStaticFile("static/dashboards.yml", filepath.Join(hubDir, "grafana", "provisioning", "dashboards", "dashboards.yml")); err != nil {
		return err
	}
	if err := writeStaticFile("static/unified-overview.json", filepath.Join(hubDir, "grafana", "dashboards", "unified-overview.json")); err != nil {
		return err
	}

	return nil
}

func renderTemplate(tmplPath, outPath string, data any) error {
	tmplData, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", outPath, err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		return fmt.Errorf("rendering %s: %w", tmplPath, err)
	}

	return f.Close()
}

func writeStaticFile(srcPath, dstPath string) error {
	data, err := staticFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading static file %s: %w", srcPath, err)
	}
	return os.WriteFile(dstPath, data, 0o644)
}
