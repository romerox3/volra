// Package doctor implements the volra doctor command logic.
package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
)

// checkResult holds the outcome of a single prerequisite check.
type checkResult struct {
	name   string
	passed bool
	detail string
	err    *output.UserError
	warn   *output.UserWarning
}

// SystemInfo abstracts system queries for testability.
type SystemInfo interface {
	PythonVersion(ctx context.Context) (string, error)
	AvailableDiskGB() (float64, error)
	IsPortFree(port int) bool
}

// defaultSystemInfo uses real system calls.
type defaultSystemInfo struct{}

func (d *defaultSystemInfo) PythonVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "python3", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *defaultSystemInfo) AvailableDiskGB() (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		return 0, err
	}
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	return float64(availableBytes) / (1024 * 1024 * 1024), nil
}

func (d *defaultSystemInfo) IsPortFree(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
	if err != nil {
		return true // connection refused or timeout = port free
	}
	_ = conn.Close()
	return false // connection succeeded = port in use
}

// Run executes all prerequisite checks and reports results via the Presenter.
// If af is provided and has observability.level >= 2, an additional Level 2 check is performed.
func Run(ctx context.Context, version string, p output.Presenter, r docker.DockerRunner, sys SystemInfo, af ...*agentfile.Agentfile) error {
	if sys == nil {
		sys = &defaultSystemInfo{}
	}

	p.Progress("Checking prerequisites...")

	results := runAllChecks(ctx, r, sys)

	// Optional Level 2 observability check.
	if len(af) > 0 && af[0] != nil && af[0].Observability != nil && af[0].Observability.Level >= 2 {
		metricsPort := 9101
		if af[0].Observability.MetricsPort > 0 {
			metricsPort = af[0].Observability.MetricsPort
		}
		results = append(results, checkLevel2Metrics(metricsPort))
	}

	failed := false
	for _, res := range results {
		if res.warn != nil {
			p.Warn(res.warn)
			continue
		}
		if res.passed {
			p.Progress(fmt.Sprintf("  ✓ %s", res.detail))
		} else {
			failed = true
			if res.err != nil {
				p.Error(res.err)
			}
		}
	}

	if version != "" {
		p.Progress(fmt.Sprintf("  Version: %s", version))
	}

	if failed {
		return &output.UserError{
			Code: "E100",
			What: "Some prerequisites are not met",
			Fix:  "Fix the issues above and run volra doctor again",
		}
	}

	p.Result("All checks passed")
	return nil
}

func runAllChecks(ctx context.Context, r docker.DockerRunner, sys SystemInfo) []checkResult {
	var results []checkResult

	results = append(results, checkDockerInstalled(ctx, r))
	results = append(results, checkDockerRunning(ctx, r))
	results = append(results, checkComposeV2(ctx, r))
	results = append(results, checkPython(ctx, sys))
	results = append(results, checkDiskSpace(sys))
	results = append(results, checkPorts(sys, 9090, 3001)...)

	return results
}

func checkDockerInstalled(ctx context.Context, r docker.DockerRunner) checkResult {
	_, err := r.Run(ctx, "--version")
	if err != nil {
		return checkResult{
			name: "docker-installed",
			err: &output.UserError{
				Code: output.CodeDockerNotInstalled,
				What: "Docker is not installed",
				Fix:  "Install Docker: https://docs.docker.com/get-docker/",
			},
		}
	}
	return checkResult{name: "docker-installed", passed: true, detail: "Docker installed"}
}

func checkDockerRunning(ctx context.Context, r docker.DockerRunner) checkResult {
	_, err := r.Run(ctx, "info")
	if err != nil {
		return checkResult{
			name: "docker-running",
			err: &output.UserError{
				Code: output.CodeDockerNotRunning,
				What: "Docker is not running",
				Fix:  "Start Docker Desktop and try again",
			},
		}
	}
	return checkResult{name: "docker-running", passed: true, detail: "Docker daemon is running"}
}

func checkComposeV2(ctx context.Context, r docker.DockerRunner) checkResult {
	_, err := r.Run(ctx, "compose", "version")
	if err != nil {
		return checkResult{
			name: "compose-v2",
			err: &output.UserError{
				Code: output.CodeComposeNotAvailable,
				What: "Docker Compose V2 not available",
				Fix:  "Update Docker Desktop or install docker-compose-plugin",
			},
		}
	}
	return checkResult{name: "compose-v2", passed: true, detail: "Docker Compose V2 available"}
}

func checkPython(ctx context.Context, sys SystemInfo) checkResult {
	versionOutput, err := sys.PythonVersion(ctx)
	if err != nil {
		return checkResult{
			name: "python",
			err: &output.UserError{
				Code: output.CodePythonNotFound,
				What: "Python 3 is not installed",
				Fix:  "Install Python >= 3.10 from https://www.python.org/downloads/",
			},
		}
	}

	if !isPythonVersionOK(versionOutput) {
		return checkResult{
			name: "python",
			err: &output.UserError{
				Code: output.CodePythonNotFound,
				What: fmt.Sprintf("Python version too old: %s", versionOutput),
				Fix:  "Install Python >= 3.10 from https://www.python.org/downloads/",
			},
		}
	}

	return checkResult{name: "python", passed: true, detail: fmt.Sprintf("%s installed", versionOutput)}
}

// isPythonVersionOK parses "Python X.Y.Z" and checks >= 3.10.
func isPythonVersionOK(versionOutput string) bool {
	// Expected format: "Python 3.10.5"
	parts := strings.Fields(versionOutput)
	if len(parts) < 2 {
		return false
	}
	version := parts[1]
	segments := strings.Split(version, ".")
	if len(segments) < 2 {
		return false
	}
	major, err := strconv.Atoi(segments[0])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(segments[1])
	if err != nil {
		return false
	}
	return major > 3 || (major == 3 && minor >= 10)
}

func checkDiskSpace(sys SystemInfo) checkResult {
	availGB, err := sys.AvailableDiskGB()
	if err != nil {
		return checkResult{
			name: "disk-space",
			err: &output.UserError{
				Code: output.CodeInsufficientDisk,
				What: "Could not check disk space",
				Fix:  "Free up disk space. Volra needs at least 1GB.",
			},
		}
	}

	if availGB < 1.0 {
		return checkResult{
			name: "disk-space",
			err: &output.UserError{
				Code: output.CodeInsufficientDisk,
				What: fmt.Sprintf("Insufficient disk space: %.1f GB available", availGB),
				Fix:  "Free up disk space. Volra needs at least 1GB.",
			},
		}
	}

	return checkResult{name: "disk-space", passed: true, detail: fmt.Sprintf("Disk space: %.0f GB available", availGB)}
}

func checkLevel2Metrics(port int) checkResult {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		return checkResult{
			name: "level2-metrics",
			warn: &output.UserWarning{
				What:     fmt.Sprintf("Level 2 metrics endpoint not reachable on port %d", port),
				Assumed:  "Level 2 observability configured but metrics endpoint not responding",
				Override: "Install volra-observe in your agent: pip install volra-observe",
			},
		}
	}
	_ = resp.Body.Close()
	return checkResult{name: "level2-metrics", passed: true, detail: fmt.Sprintf("Level 2 metrics endpoint responding on port %d", port)}
}

func checkPorts(sys SystemInfo, ports ...int) []checkResult {
	var results []checkResult
	for _, port := range ports {
		if !sys.IsPortFree(port) {
			results = append(results, checkResult{
				name: fmt.Sprintf("port-%d", port),
				warn: &output.UserWarning{
					What:     fmt.Sprintf("Port %d is already in use", port),
					Assumed:  "",
					Override: fmt.Sprintf("Stop the process using port %d or change port in Agentfile", port),
				},
			})
		}
	}
	return results
}
