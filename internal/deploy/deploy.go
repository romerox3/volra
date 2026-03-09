// Package deploy implements the volra deploy command logic.
package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/alerting"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
)

// Run executes the full deploy pipeline:
// Load Agentfile → Validate .env → Build context → Generate artifacts → Orchestrate → Health check → Summary.
func Run(ctx context.Context, dir string, p output.Presenter, dr docker.DockerRunner) error {
	// 1. Load Agentfile
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return err
	}
	p.Progress(fmt.Sprintf("Loaded Agentfile: %s", af.Name))

	// 2. Validate .env exists if Agentfile declares env vars (agent or any service)
	if NeedsEnv(af) {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return &output.UserError{
				Code: output.CodeEnvNotFound,
				What: "Agentfile declares env vars but .env file not found",
				Fix:  "Create .env from .env.example: cp .env.example .env",
			}
		}
	}

	// 3. Build template context
	tc := BuildContext(af, dir)

	// 4. Generate all artifacts
	p.Progress("Generating deploy artifacts...")
	if err := GenerateAll(af, tc, dir); err != nil {
		return err
	}
	p.Progress("Artifacts generated in .volra/")

	// 5. GPU pre-flight check
	if af.GPU {
		if err := CheckGPUAvailable(ctx, dr); err != nil {
			return err
		}
	}

	// 6. Orchestrate: docker compose up
	p.Progress("Starting services...")
	if err := Orchestrate(ctx, dr, af.Name, dir); err != nil {
		return err
	}

	// 7. Health check
	healthTimeout := time.Duration(af.HealthTimeout) * time.Second
	if err := WaitForHealth(ctx, tc.AgentHostPort, af.HealthPath, af.Name, healthTimeout, p); err != nil {
		return err
	}

	// 8. Register in global agent registry
	if err := registry.Register(af.Name, dir, tc.PrometheusHostPort, tc.AgentHostPort); err != nil {
		p.Warn(&output.UserWarning{
			What:    fmt.Sprintf("Could not register agent in registry: %v", err),
			Assumed: "Agent deployed but not visible in `volra hub`",
		})
	}

	// 9. Summary
	p.Result(fmt.Sprintf("Agent:      http://localhost:%d", tc.AgentHostPort))
	p.Result(fmt.Sprintf("Grafana:    http://localhost:%d", tc.GrafanaHostPort))
	p.Result(fmt.Sprintf("Prometheus: http://localhost:%d", tc.PrometheusHostPort))
	if af.Alerts != nil {
		p.Result(fmt.Sprintf("Alertmanager: http://localhost:%d", tc.AlertmanagerHostPort))
	}
	p.Result(fmt.Sprintf("Stop:       docker compose -f %s/docker-compose.yml -p %s down", filepath.Join(dir, OutputDir), af.Name))

	return nil
}

// NeedsEnv returns true if the Agentfile declares env vars (agent or any service).
func NeedsEnv(af *agentfile.Agentfile) bool {
	if len(af.Env) > 0 {
		return true
	}
	for _, svc := range af.Services {
		if len(svc.Env) > 0 {
			return true
		}
	}
	return false
}

// GenerateAll generates all deployment artifacts (Dockerfile, docker-compose, Prometheus, Grafana, env files).
// Extracted for reuse by volra dev.
func GenerateAll(af *agentfile.Agentfile, tc *TemplateContext, dir string) error {
	if af.Dockerfile == agentfile.DockerfileModeAuto {
		if err := GenerateDockerfile(tc, dir); err != nil {
			return fmt.Errorf("generating Dockerfile: %w", err)
		}
	}
	if err := GenerateCompose(tc, dir); err != nil {
		return fmt.Errorf("generating docker-compose.yml: %w", err)
	}
	if err := GeneratePrometheus(tc, dir); err != nil {
		return fmt.Errorf("generating prometheus.yml: %w", err)
	}
	if err := CopyAlertRules(dir); err != nil {
		return fmt.Errorf("copying alert_rules.yml: %w", err)
	}
	if err := CopyBlackboxConfig(dir); err != nil {
		return fmt.Errorf("copying blackbox.yml: %w", err)
	}
	if err := CopyGrafanaAssets(dir, tc.HasMetrics, tc.HasLevel2); err != nil {
		return fmt.Errorf("copying Grafana assets: %w", err)
	}

	if NeedsEnv(af) {
		if err := GenerateEnvFiles(af, dir); err != nil {
			return fmt.Errorf("generating env files: %w", err)
		}
	}

	if af.Alerts != nil {
		if err := GenerateAlertingConfigs(af.Alerts, dir); err != nil {
			return fmt.Errorf("generating alerting configs: %w", err)
		}
	}

	return nil
}

// GenerateAlertingConfigs renders alertmanager.yml and alert-rules.yml into the output dir.
func GenerateAlertingConfigs(alerts *agentfile.AlertsConfig, dir string) error {
	outputDir := filepath.Join(dir, OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	amConfig, err := alerting.RenderAlertmanagerConfig(alerts)
	if err != nil {
		return fmt.Errorf("rendering alertmanager config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "alertmanager.yml"), []byte(amConfig), 0644); err != nil {
		return fmt.Errorf("writing alertmanager.yml: %w", err)
	}

	rulesConfig, err := alerting.RenderAlertRules(alerts)
	if err != nil {
		return fmt.Errorf("rendering alert rules: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "alert-rules.yml"), []byte(rulesConfig), 0644); err != nil {
		return fmt.Errorf("writing alert-rules.yml: %w", err)
	}

	return nil
}
