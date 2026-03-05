// Package deploy implements the volra deploy command logic.
package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
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
	needsEnv := len(af.Env) > 0
	if !needsEnv {
		for _, svc := range af.Services {
			if len(svc.Env) > 0 {
				needsEnv = true
				break
			}
		}
	}
	if needsEnv {
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
	if needsEnv {
		if err := GenerateEnvFiles(af, dir); err != nil {
			return fmt.Errorf("generating env files: %w", err)
		}
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

	// 8. Summary
	p.Result(fmt.Sprintf("Agent:      http://localhost:%d", tc.AgentHostPort))
	p.Result(fmt.Sprintf("Grafana:    http://localhost:%d", tc.GrafanaHostPort))
	p.Result(fmt.Sprintf("Prometheus: http://localhost:%d", tc.PrometheusHostPort))
	p.Result(fmt.Sprintf("Stop:       docker compose -f %s/docker-compose.yml -p %s down", filepath.Join(dir, OutputDir), af.Name))

	return nil
}
