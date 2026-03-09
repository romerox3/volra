package deploy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/romerox3/volra/internal/agentfile"
)

// VolumeSpec represents a named Docker volume and its container mount path.
type VolumeSpec struct {
	Name      string // Docker named volume, e.g., "my-agent-data"
	MountPath string // Container mount path, e.g., "/data"
}

// ServiceContext is the pre-computed deploy-time representation of a Service.
type ServiceContext struct {
	Name        string       // map key, used as compose service name suffix
	Image       string
	Port        int
	HostPort    int          // 0 = no host port mapping
	Env         []string
	VolumeSpecs []VolumeSpec
	Healthcheck *DeployHealthcheck
	Resources   *DeployResources
}

// TemplateContext extends Agentfile with deploy-time detected metadata.
type TemplateContext struct {
	agentfile.Agentfile
	PythonVersion      string
	EntryPoint         string
	PackageManager     string // "pip", "poetry", "uv", "pipenv"
	HasRequirements    bool   // true when requirements.txt exists (used for pip branch)
	HasMetrics         bool
	VolumeSpecs        []VolumeSpec
	ServiceContexts    []ServiceContext
	AgentHostPort      int // resolved: host_port || port
	PrometheusHostPort int
	GrafanaHostPort    int
	// SecurityTmpfs holds the resolved tmpfs mounts (auto-injected if read_only + no explicit tmpfs).
	SecurityTmpfs []agentfile.TmpfsMount
	// Observability Level 2 fields.
	ObservabilityLevel       int
	ObservabilityMetricsPort int
	HasLevel2                bool
	AlertmanagerHostPort     int
}

// JobHealth returns the Prometheus job name for health endpoint scraping.
func (tc *TemplateContext) JobHealth() string { return JobHealth }

// JobMetrics returns the Prometheus job name for metrics scraping.
func (tc *TemplateContext) JobMetrics() string { return JobMetrics }

// entryPointCandidates in priority order for deploy-time detection.
var deployEntryPointCandidates = []string{"main.py", "app.py", "server.py"}

// BuildContext creates a TemplateContext by combining Agentfile with deploy-time detection.
func BuildContext(af *agentfile.Agentfile, dir string) *TemplateContext {
	// Resolve host ports
	agentHostPort := af.Port
	if af.HostPort > 0 {
		agentHostPort = af.HostPort
	}

	// Resolve observability
	obsLevel := 1
	obsMetricsPort := 9101
	if af.Observability != nil {
		if af.Observability.Level > 0 {
			obsLevel = af.Observability.Level
		}
		if af.Observability.MetricsPort > 0 {
			obsMetricsPort = af.Observability.MetricsPort
		}
	}

	tc := &TemplateContext{
		Agentfile:                *af,
		PythonVersion:            detectPythonVersion(dir),
		EntryPoint:               detectDeployEntryPoint(dir),
		PackageManager:           detectDeployPackageManager(af, dir),
		HasMetrics:               detectMetricsLibrary(dir),
		VolumeSpecs:              buildVolumeSpecs(af.Name, af.Volumes),
		ServiceContexts:          buildServiceContexts(af.Name, af.Services),
		AgentHostPort:            agentHostPort,
		PrometheusHostPort:       9090,
		GrafanaHostPort:          3001,
		ObservabilityLevel:       obsLevel,
		ObservabilityMetricsPort: obsMetricsPort,
		HasLevel2:                obsLevel >= 2,
		AlertmanagerHostPort:     9093,
	}
	_, err := os.Stat(filepath.Join(dir, "requirements.txt"))
	tc.HasRequirements = err == nil

	// Auto-inject tmpfs for read_only containers
	if af.Security != nil && af.Security.ReadOnly {
		if len(af.Security.Tmpfs) > 0 {
			tc.SecurityTmpfs = af.Security.Tmpfs
		} else {
			tc.SecurityTmpfs = []agentfile.TmpfsMount{
				{Path: "/tmp", Size: "100M"},
				{Path: "/app/__pycache__", Size: "50M"},
			}
		}
	} else if af.Security != nil && len(af.Security.Tmpfs) > 0 {
		tc.SecurityTmpfs = af.Security.Tmpfs
	}

	return tc
}

// detectPythonVersion tries to extract Python version from pyproject.toml, falls back to "3.11".
func detectPythonVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return "3.11"
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "requires-python") {
			// Extract version like >=3.12 → 3.12
			for i, c := range trimmed {
				if c >= '0' && c <= '9' {
					ver := strings.TrimSpace(trimmed[i:])
					ver = strings.Trim(ver, "\"'")
					// Take major.minor only
					parts := strings.SplitN(ver, ".", 3)
					if len(parts) >= 2 {
						return parts[0] + "." + parts[1]
					}
					return ver
				}
			}
		}
	}
	return "3.11"
}

// detectDeployEntryPoint finds the entry point at deploy time.
func detectDeployEntryPoint(dir string) string {
	for _, candidate := range deployEntryPointCandidates {
		if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
			return candidate
		}
	}
	return "main.py"
}

// buildVolumeSpecs converts container mount paths into Docker named volume specs.
// Naming convention: {agentName}-{path-segments-joined-by-dashes}
// Example: agentName="my-agent", path="/data/models" → name="my-agent-data-models"
func buildVolumeSpecs(agentName string, volumes []string) []VolumeSpec {
	specs := make([]VolumeSpec, 0, len(volumes))
	for _, v := range volumes {
		name := agentName + "-" + strings.TrimPrefix(v, "/")
		name = strings.ReplaceAll(name, "/", "-")
		specs = append(specs, VolumeSpec{Name: name, MountPath: v})
	}
	return specs
}

// buildServiceContexts converts the Services map into a sorted slice of ServiceContexts.
// Sorted by name for deterministic compose output (Go map iteration is non-deterministic).
func buildServiceContexts(agentName string, services map[string]agentfile.Service) []ServiceContext {
	if len(services) == 0 {
		return nil
	}
	contexts := make([]ServiceContext, 0, len(services))
	for name, svc := range services {
		sc := ServiceContext{
			Name:        name,
			Image:       svc.Image,
			Port:        svc.Port,
			HostPort:    svc.HostPort,
			Env:         svc.Env,
			VolumeSpecs: buildVolumeSpecs(agentName+"-"+name, svc.Volumes),
			Healthcheck: resolveHealthcheck(svc.Image, svc.Healthcheck),
			Resources:   resolveResources(svc.Image, svc.Resources),
		}
		contexts = append(contexts, sc)
	}
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Name < contexts[j].Name
	})
	return contexts
}

// detectDeployPackageManager resolves the package manager for Dockerfile generation.
// If the Agentfile specifies a package_manager, that value is used.
// Otherwise, auto-detect by checking for lockfiles (same priority as setup scanner).
func detectDeployPackageManager(af *agentfile.Agentfile, dir string) string {
	if af.PackageManager != "" {
		return string(af.PackageManager)
	}
	// Auto-detect by lockfile priority: uv.lock > poetry.lock > Pipfile.lock > pip
	if _, err := os.Stat(filepath.Join(dir, "uv.lock")); err == nil {
		return "uv"
	}
	if _, err := os.Stat(filepath.Join(dir, "poetry.lock")); err == nil {
		return "poetry"
	}
	if _, err := os.Stat(filepath.Join(dir, "Pipfile.lock")); err == nil {
		return "pipenv"
	}
	return "pip"
}

// detectMetricsLibrary checks if the project uses prometheus_client for custom metrics.
func detectMetricsLibrary(dir string) bool {
	for _, file := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if data, err := os.ReadFile(filepath.Join(dir, file)); err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "prometheus_client") || strings.Contains(content, "prometheus-client") {
				return true
			}
		}
	}
	return false
}
