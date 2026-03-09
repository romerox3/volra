// Package dev implements the volra dev command logic (hot-reload development mode).
package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/doctor"
	"github.com/romerox3/volra/internal/output"
)

// WatchExecutor runs docker compose watch. Abstracted for testability.
type WatchExecutor func(ctx context.Context, composePath, projectName string) error

// ComposeVersionChecker checks the Docker Compose version for watch support.
type ComposeVersionChecker func(ctx context.Context, dr docker.DockerRunner) (string, error)

// DefaultComposeVersionChecker returns the compose version using --short flag.
func DefaultComposeVersionChecker(ctx context.Context, dr docker.DockerRunner) (string, error) {
	out, err := dr.Run(ctx, "compose", "version", "--short")
	if err != nil {
		return "", fmt.Errorf("checking compose version: %w", err)
	}
	return out, nil
}

// Run starts the dev loop:
// 1. Load Agentfile
// 2. Verify Compose >= 2.22
// 3. Generate artifacts (reuses deploy.GenerateAll)
// 4. Execute docker compose watch with stdout/stderr streaming
func Run(ctx context.Context, dir string, p output.Presenter, dr docker.DockerRunner, checker ComposeVersionChecker, executor WatchExecutor) error {
	// 1. Load Agentfile
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return err
	}
	p.Progress(fmt.Sprintf("Loaded Agentfile: %s", af.Name))

	// 2. Verify Compose >= 2.22 for watch support
	if checker == nil {
		checker = DefaultComposeVersionChecker
	}
	version, err := checker(ctx, dr)
	if err != nil {
		return &output.UserError{
			Code: output.CodeComposeWatchRequired,
			What: "Could not determine Docker Compose version",
			Fix:  "Install Docker Compose >= 2.22.0 for volra dev support",
		}
	}
	if !IsComposeWatchSupported(version) {
		return &output.UserError{
			Code: output.CodeComposeWatchRequired,
			What: fmt.Sprintf("Docker Compose %s does not support watch (requires >= 2.22.0)", version),
			Fix:  "Update Docker Desktop or install docker-compose-plugin >= 2.22.0",
		}
	}

	// 3. Validate .env if needed
	if deploy.NeedsEnv(af) {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return &output.UserError{
				Code: output.CodeEnvNotFound,
				What: "Agentfile declares env vars but .env file not found",
				Fix:  "Create .env from .env.example: cp .env.example .env",
			}
		}
	}

	// 4. Generate artifacts
	tc := deploy.BuildContext(af, dir)
	p.Progress("Generating deploy artifacts...")
	if err := deploy.GenerateAll(af, tc, dir); err != nil {
		return err
	}
	p.Progress("Artifacts generated in .volra/")

	// 5. Start compose watch (foreground, streaming)
	composePath := filepath.Join(dir, deploy.OutputDir, "docker-compose.yml")
	p.Progress("Starting development mode (Ctrl+C to stop)...")

	if executor == nil {
		executor = defaultWatchExecutor
	}
	return executor(ctx, composePath, af.Name)
}

// defaultWatchExecutor runs docker compose watch with stdout/stderr streaming.
func defaultWatchExecutor(ctx context.Context, composePath, projectName string) error {
	args := []string{"compose", "-f", composePath, "-p", projectName, "watch"}
	c := exec.CommandContext(ctx, "docker", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// IsComposeWatchSupported checks if a compose version string is >= 2.22.0.
func IsComposeWatchSupported(version string) bool {
	version = strings.TrimSpace(version)
	return doctor.IsComposeVersionAtLeast(version, 2, 22, 0)
}
