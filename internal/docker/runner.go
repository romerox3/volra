// Package docker provides the DockerRunner interface for all Docker interactions.
package docker

import (
	"context"
	"os/exec"
)

// DockerRunner abstracts Docker CLI interactions for testability.
type DockerRunner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

// ExecRunner implements DockerRunner via os/exec.
type ExecRunner struct{}

// NewExecRunner creates a new ExecRunner.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{}
}

// Run executes a docker command with the given arguments.
func (e *ExecRunner) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
