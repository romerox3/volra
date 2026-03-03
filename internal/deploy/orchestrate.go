package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/antonioromero/volra/internal/docker"
	"github.com/antonioromero/volra/internal/output"
)

// Orchestrate runs docker compose up to start all services.
func Orchestrate(ctx context.Context, dr docker.DockerRunner, name string, dir string) error {
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")

	out, err := dr.Run(ctx,
		"compose", "-f", composePath, "-p", name, "up", "-d", "--build",
	)
	if err != nil {
		return classifyComposeError(out, err)
	}
	return nil
}

// classifyComposeError maps docker compose output to user-facing errors.
func classifyComposeError(out string, err error) error {
	lower := strings.ToLower(out)

	if strings.Contains(lower, "cannot connect to the docker daemon") ||
		strings.Contains(lower, "is the docker daemon running") {
		return &output.UserError{
			Code: output.CodeDeployDockerNotRunning,
			What: "Docker is not running",
			Fix:  "Start Docker Desktop or the Docker daemon, then retry: volra deploy",
		}
	}

	if strings.Contains(lower, "build") && (strings.Contains(lower, "failed") || strings.Contains(lower, "error")) {
		excerpt := extractExcerpt(out, 5)
		return &output.UserError{
			Code: output.CodeBuildFailed,
			What: fmt.Sprintf("Agent build failed:\n%s", excerpt),
			Fix:  "Check your Dockerfile / requirements.txt for errors, then retry: volra deploy",
		}
	}

	// Fallback: return original error wrapped
	return fmt.Errorf("docker compose up failed: %w\n%s", err, out)
}

// extractExcerpt returns the last n lines of output as a build log excerpt.
func extractExcerpt(out string, n int) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) <= n {
		return strings.TrimSpace(out)
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
