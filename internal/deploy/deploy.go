// Package deploy implements the volra deploy command logic.
package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/antonioromero/volra/internal/agentfile"
	"github.com/antonioromero/volra/internal/docker"
	"github.com/antonioromero/volra/internal/output"
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

	// 2. Validate .env exists if Agentfile declares env vars
	if len(af.Env) > 0 {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return &output.UserError{
				Code: output.CodeEnvNotFound,
				What: fmt.Sprintf("Agentfile declares env vars %v but .env file not found", af.Env),
				Fix:  "Create .env from .env.example: cp .env.example .env",
			}
		}
	}

	// 3. Build template context
	tc := BuildContext(af, dir)

	// 4. Generate all artifacts
	p.Progress("Generating deploy artifacts...")
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
	if err := CopyGrafanaAssets(dir); err != nil {
		return fmt.Errorf("copying Grafana assets: %w", err)
	}
	p.Progress("Artifacts generated in .volra/")

	// 5. Orchestrate: docker compose up
	p.Progress("Starting services...")
	if err := Orchestrate(ctx, dr, af.Name, dir); err != nil {
		return err
	}

	// 6. Health check
	if err := WaitForHealth(ctx, af.Port, af.HealthPath, af.Name, p); err != nil {
		return err
	}

	// 7. Summary
	p.Result(fmt.Sprintf("Agent:      http://localhost:%d", af.Port))
	p.Result("Grafana:    http://localhost:3001")
	p.Result("Prometheus: http://localhost:9090")
	p.Result(fmt.Sprintf("Stop:       docker compose -f %s/docker-compose.yml -p %s down", filepath.Join(dir, OutputDir), af.Name))

	return nil
}
