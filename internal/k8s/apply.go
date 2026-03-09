package k8s

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CheckKubectl verifies that kubectl is available and the cluster is reachable.
func CheckKubectl(ctx context.Context) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found in PATH: install from https://kubernetes.io/docs/tasks/tools/")
	}

	cmd := exec.CommandContext(ctx, "kubectl", "cluster-info", "--request-timeout=5s")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kubectl cluster unreachable: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Apply runs kubectl apply on the generated manifests directory.
func Apply(ctx context.Context, manifestDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl apply failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// DryRun runs kubectl apply --dry-run=client on the manifests.
func DryRun(ctx context.Context, manifestDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestDir, "--dry-run=client")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl dry-run failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
