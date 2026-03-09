package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/k8s"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var (
	dryRun       bool
	deployTarget string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [path]",
	Short: "Deploy agent with monitoring stack",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		p := newPresenter()
		defer flushPresenter(p)

		if dryRun {
			return runDryRun(dir, p)
		}

		if deployTarget == "k8s" {
			return runDeployK8s(cmd, dir, p)
		}

		dr := docker.NewExecRunner()
		return deploy.Run(cmd.Context(), dir, p, dr)
	},
}

func init() {
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without deploying")
	deployCmd.Flags().StringVar(&deployTarget, "target", "docker", "Deploy target: docker, k8s")
	rootCmd.AddCommand(deployCmd)
}

func runDeployK8s(cmd *cobra.Command, dir string, p output.Presenter) error {
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return err
	}
	p.Progress(fmt.Sprintf("Loaded Agentfile: %s (target: k8s)", af.Name))

	// Check kubectl availability.
	p.Progress("Checking kubectl...")
	if err := k8s.CheckKubectl(cmd.Context()); err != nil {
		return &output.UserError{
			Code: output.CodeKubectlNotFound,
			What: err.Error(),
			Fix:  "Install kubectl and configure cluster access",
		}
	}

	// Build K8s manifest context from Agentfile.
	mctx := &k8s.ManifestContext{
		Name:       af.Name,
		Image:      fmt.Sprintf("%s:latest", af.Name),
		Port:       af.Port,
		HealthPath: af.HealthPath,
		Replicas:   1,
	}
	for _, name := range af.Env {
		mctx.Env = append(mctx.Env, k8s.EnvVar{Name: name, Value: ""})
	}
	for _, vol := range af.Volumes {
		parts := strings.SplitN(vol, ":", 2)
		name := strings.ReplaceAll(filepath.Base(parts[0]), "/", "-")
		mount := parts[0]
		if len(parts) == 2 {
			mount = parts[1]
		}
		mctx.Volumes = append(mctx.Volumes, k8s.VolumeMount{Name: name, MountPath: mount})
	}

	// Generate manifests.
	p.Progress("Generating Kubernetes manifests...")
	if err := k8s.GenerateAll(mctx, dir); err != nil {
		return &output.UserError{
			Code: output.CodeK8sManifestFailed,
			What: fmt.Sprintf("Failed to generate manifests: %v", err),
			Fix:  "Check Agentfile configuration",
		}
	}

	// Apply manifests.
	manifestDir := filepath.Join(dir, ".volra", "k8s")
	p.Progress("Applying manifests with kubectl...")
	result, err := k8s.Apply(cmd.Context(), manifestDir)
	if err != nil {
		return &output.UserError{
			Code: output.CodeKubectlApplyFailed,
			What: err.Error(),
			Fix:  "Check kubectl configuration and cluster access",
		}
	}

	p.Result(fmt.Sprintf("Kubernetes deploy complete:\n%s", result))
	return nil
}

func runDryRun(dir string, p output.Presenter) error {
	// 1. Load Agentfile
	agentfilePath := filepath.Join(dir, "Agentfile")
	af, err := agentfile.Load(agentfilePath)
	if err != nil {
		return err
	}
	p.Progress(fmt.Sprintf("Loaded Agentfile: %s (dry-run)", af.Name))

	// 2. Validate .env if needed
	if deploy.NeedsEnv(af) {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return fmt.Errorf("Agentfile declares env vars but .env file not found — create .env from .env.example")
		}
	}

	// 3. Build context
	tc := deploy.BuildContext(af, dir)

	// 4. Generate artifacts to temp dir
	tempDir, err := os.MkdirTemp("", "volra-dry-run-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	p.Progress("Generating artifacts to temp directory...")
	if err := deploy.GenerateAll(af, tc, tempDir); err != nil {
		return err
	}

	// 4. Compare with existing .volra/
	currentDir := filepath.Join(dir, deploy.OutputDir)
	tempOutputDir := filepath.Join(tempDir, deploy.OutputDir)

	if _, err := os.Stat(currentDir); os.IsNotExist(err) {
		// No existing deployment — show all as new
		p.Result("No existing deployment found. All files would be new:")
		return showNewFiles(tempOutputDir, p)
	}

	// 5. Run diff
	p.Progress("Comparing generated artifacts against .volra/...")
	diffOutput, changed, err := runDiff(currentDir, tempOutputDir)
	if err != nil {
		return err
	}

	if !changed {
		p.Result("No changes detected")
		return nil
	}

	p.Result(diffOutput)
	return nil
}

func showNewFiles(dir string, p output.Presenter) error {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, "  + "+rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.Result(strings.Join(files, "\n"))
	return nil
}

func runDiff(currentDir, generatedDir string) (string, bool, error) {
	cmd := exec.Command("diff", "-ru", currentDir, generatedDir)
	out, err := cmd.CombinedOutput()

	if err != nil {
		// diff returns exit code 1 when files differ — that's expected
		if cmd.ProcessState.ExitCode() == 1 {
			return string(out), true, nil
		}
		return "", false, fmt.Errorf("diff failed: %w\n%s", err, out)
	}

	// Exit code 0 = no differences
	return "", false, nil
}
