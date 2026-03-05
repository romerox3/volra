package deploy

import (
	"context"
	"strings"

	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
)

// CheckGPUAvailable verifies that the NVIDIA Container Runtime is available.
func CheckGPUAvailable(ctx context.Context, dr docker.DockerRunner) error {
	out, err := dr.Run(ctx, "info", "--format", "{{.Runtimes}}")
	if err != nil {
		return &output.UserError{
			Code: output.CodeGPUCheckFailed,
			What: "Failed to check GPU availability: " + err.Error(),
			Fix:  "Ensure Docker is running, then retry: volra deploy",
		}
	}
	if !strings.Contains(out, "nvidia") {
		return &output.UserError{
			Code: output.CodeGPUNotAvailable,
			What: "GPU deploy requested but NVIDIA Container Runtime not found",
			Fix:  "Install nvidia-container-toolkit, then retry: volra deploy",
		}
	}
	return nil
}
