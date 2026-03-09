package agentfile

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/romerox3/volra/internal/output"
)

const supportedVersion = 1

var dnsLabelRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

// Validate checks semantic rules on a parsed Agentfile.
func Validate(af *Agentfile) error {
	if err := validateVersion(af.Version); err != nil {
		return err
	}
	if err := validateName(af.Name); err != nil {
		return err
	}
	if err := validatePort(af.Port); err != nil {
		return err
	}
	if err := validateHealthPath(af.HealthPath); err != nil {
		return err
	}
	if err := validateHealthTimeout(af.HealthTimeout); err != nil {
		return err
	}
	if err := validateVolumes(af.Volumes); err != nil {
		return err
	}
	if err := validateEnv(af.Env); err != nil {
		return err
	}
	if err := validateServices(af.Services, af.Port); err != nil {
		return err
	}
	if err := validateSecurity(af.Security); err != nil {
		return err
	}
	if err := validateBuild(af); err != nil {
		return err
	}
	if err := validateHostPort(af); err != nil {
		return err
	}
	if err := validateObservability(af.Observability); err != nil {
		return err
	}
	if err := validateEval(af.Eval); err != nil {
		return err
	}
	return nil
}

func validateVersion(v int) error {
	if v == 0 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: version — required",
			Fix:  "Add 'version: 1' to your Agentfile",
		}
	}
	if v > supportedVersion {
		return &output.UserError{
			Code: output.CodeUnsupportedVersion,
			What: fmt.Sprintf("Unsupported Agentfile version %d", v),
			Fix:  fmt.Sprintf("This Volra version supports Agentfile version %d. Update Volra or downgrade your Agentfile.", supportedVersion),
		}
	}
	return nil
}

func validateName(name string) error {
	if name == "" {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: name — required",
			Fix:  "Add 'name' to your Agentfile",
		}
	}
	if len(name) < 2 || len(name) > 63 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: name — must be 2-63 characters, got %d", len(name)),
			Fix:  "Use a name between 2 and 63 characters",
		}
	}
	if !dnsLabelRegex.MatchString(name) {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: name — %q is not a valid DNS label", name),
			Fix:  "Use only lowercase letters, digits, and hyphens. Must start with a letter and end with a letter or digit.",
		}
	}
	return nil
}

func validatePort(port int) error {
	if port == 0 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: port — required",
			Fix:  "Add 'port: 8000' to your Agentfile",
		}
	}
	if port < 1 || port > 65535 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: port — %d is out of range", port),
			Fix:  "Use a port between 1 and 65535",
		}
	}
	return nil
}

func validateHealthPath(path string) error {
	if path == "" {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: health_path — required",
			Fix:  "Add 'health_path: /health' to your Agentfile",
		}
	}
	if !strings.HasPrefix(path, "/") {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: health_path — %q must start with /", path),
			Fix:  "Prefix your health path with /",
		}
	}
	return nil
}

func validateHealthTimeout(timeout int) error {
	if timeout == 0 {
		return nil // not specified, use default
	}
	if timeout < 10 || timeout > 600 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: health_timeout — %d is out of range (10-600)", timeout),
			Fix:  "Set health_timeout between 10 and 600 seconds, or remove the field for default (60s)",
		}
	}
	return nil
}

func validateVolumes(volumes []string) error {
	if len(volumes) > 10 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: volumes — too many entries (max 10)",
			Fix:  "Reduce the number of volume mounts to 10 or fewer",
		}
	}
	seen := make(map[string]bool)
	for _, v := range volumes {
		if v == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: volumes — contains empty entry",
				Fix:  "Remove empty entries from the volumes list",
			}
		}
		if !strings.HasPrefix(v, "/") {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: volumes — %q must be an absolute path", v),
				Fix:  "Use absolute paths starting with / for volume mounts",
			}
		}
		if v == "/" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: volumes — cannot mount at root /",
				Fix:  "Choose a subdirectory like /data or /models",
			}
		}
		if v == "/app" || strings.HasPrefix(v, "/app/") {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: volumes — %q conflicts with container WORKDIR /app", v),
				Fix:  "Choose a different mount path — /app is reserved for agent code",
			}
		}
		if seen[v] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: volumes — duplicate volume path %q", v),
				Fix:  fmt.Sprintf("Remove the duplicate %q from the volumes list", v),
			}
		}
		seen[v] = true
	}
	return nil
}

var reservedServiceNames = map[string]bool{
	"agent":      true,
	"prometheus": true,
	"grafana":    true,
	"blackbox":   true,
}

func validateServices(services map[string]Service, agentPort int) error {
	if len(services) > 5 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: services — too many services (max 5)",
			Fix:  "Reduce the number of services to 5 or fewer",
		}
	}
	// Sort service names for deterministic error messages.
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		svc := services[name]
		if len(name) < 2 || len(name) > 63 || !dnsLabelRegex.MatchString(name) {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services — %q is not a valid DNS label", name),
				Fix:  "Use only lowercase letters, digits, and hyphens. Must start with a letter and end with a letter or digit.",
			}
		}
		if reservedServiceNames[name] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services — %q is a reserved service name", name),
				Fix:  "Choose a different name. Reserved: agent, prometheus, grafana, blackbox",
			}
		}
		if svc.Image == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services.%s — image is required", name),
				Fix:  fmt.Sprintf("Add 'image' to the %s service definition (e.g., redis:7-alpine)", name),
			}
		}
		if svc.Port != 0 {
			if svc.Port < 1 || svc.Port > 65535 {
				return &output.UserError{
					Code: output.CodeInvalidAgentfile,
					What: fmt.Sprintf("Invalid field: services.%s — port %d is out of range", name, svc.Port),
					Fix:  "Use a port between 1 and 65535",
				}
			}
		}
		// Only host_ports can conflict with agent port (container ports are isolated).
		// Inter-service host_port uniqueness is validated in validateHostPort().
		if svc.HostPort != 0 && svc.HostPort == agentPort {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services.%s — host_port %d conflicts with agent port", name, svc.HostPort),
				Fix:  "Choose a different host_port for this service",
			}
		}
		if err := validateServiceVolumes(svc.Volumes); err != nil {
			return err
		}
		if err := validateEnv(svc.Env); err != nil {
			return err
		}
	}
	return nil
}

func validateServiceVolumes(volumes []string) error {
	seen := make(map[string]bool)
	for _, v := range volumes {
		if v == "" || !strings.HasPrefix(v, "/") || v == "/" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services — volume %q must be an absolute path", v),
				Fix:  "Use absolute paths starting with / for volume mounts",
			}
		}
		if seen[v] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: services — duplicate volume path %q", v),
				Fix:  fmt.Sprintf("Remove the duplicate %q", v),
			}
		}
		seen[v] = true
	}
	return nil
}

var capabilityRegex = regexp.MustCompile(`^[A-Z][A-Z_]*$`)

func validateSecurity(sec *SecurityContext) error {
	if sec == nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, cap := range sec.DropCapabilities {
		if cap == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: security.drop_capabilities — contains empty entry",
				Fix:  "Remove empty entries from the drop_capabilities list",
			}
		}
		if !capabilityRegex.MatchString(cap) {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: security.drop_capabilities — %q is not a valid Linux capability", cap),
				Fix:  "Use uppercase capability names like ALL, NET_RAW, SYS_ADMIN",
			}
		}
		if seen[cap] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: security.drop_capabilities — duplicate entry %q", cap),
				Fix:  fmt.Sprintf("Remove the duplicate %q from the drop_capabilities list", cap),
			}
		}
		seen[cap] = true
	}
	return nil
}

func validateEnv(env []string) error {
	seen := make(map[string]bool)
	for _, e := range env {
		if e == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: env — contains empty entry",
				Fix:  "Remove empty entries from the env list",
			}
		}
		if seen[e] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: env — duplicate entry %q", e),
				Fix:  fmt.Sprintf("Remove the duplicate %q from the env list", e),
			}
		}
		seen[e] = true
	}
	return nil
}

func validateBuild(af *Agentfile) error {
	if af.Build == nil {
		return nil
	}
	if af.Dockerfile != DockerfileModeAuto {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: build — only valid with dockerfile: auto",
			Fix:  "Remove the build section or set dockerfile: auto",
		}
	}
	if len(af.Build.SetupCommands) > 20 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: build.setup_commands — too many commands (max 20)",
			Fix:  "Reduce the number of setup commands to 20 or fewer",
		}
	}
	for _, cmd := range af.Build.SetupCommands {
		if strings.TrimSpace(cmd) == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: build.setup_commands — contains empty command",
				Fix:  "Remove empty entries from setup_commands",
			}
		}
	}
	if len(af.Build.CacheDirs) > 10 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: build.cache_dirs — too many entries (max 10)",
			Fix:  "Reduce the number of cache dirs to 10 or fewer",
		}
	}
	for _, d := range af.Build.CacheDirs {
		if !strings.HasPrefix(d, "/") {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: build.cache_dirs — %q must be an absolute path", d),
				Fix:  "Use absolute paths starting with / for cache dirs",
			}
		}
		if d == "/" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: "Invalid field: build.cache_dirs — cannot use root /",
				Fix:  "Choose a subdirectory like /root/.cache",
			}
		}
	}
	return nil
}

func validateObservability(obs *ObservabilityConfig) error {
	if obs == nil {
		return nil
	}
	if obs.Level != 0 && obs.Level != 1 && obs.Level != 2 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: observability.level — must be 1 or 2, got %d", obs.Level),
			Fix:  "Set observability.level to 1 (probe only) or 2 (LLM metrics)",
		}
	}
	if obs.MetricsPort != 0 {
		if obs.MetricsPort < 1 || obs.MetricsPort > 65535 {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: observability.metrics_port — %d is out of range", obs.MetricsPort),
				Fix:  "Use a port between 1 and 65535",
			}
		}
	}
	return nil
}

func validateEval(eval *EvalConfig) error {
	if eval == nil {
		return nil
	}
	if len(eval.Metrics) == 0 {
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: "Invalid field: eval.metrics — at least one metric is required",
			Fix:  "Add metrics to the eval section with name, query, and threshold",
		}
	}
	seen := make(map[string]bool)
	for i, m := range eval.Metrics {
		if m.Name == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: eval.metrics[%d].name — required", i),
				Fix:  "Add a name to each eval metric",
			}
		}
		if m.Query == "" {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: eval.metrics[%d].query — required", i),
				Fix:  fmt.Sprintf("Add a PromQL query for metric %q", m.Name),
			}
		}
		if m.Threshold <= 0 {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: eval.metrics[%d].threshold — must be > 0, got %d", i, m.Threshold),
				Fix:  fmt.Sprintf("Set a positive threshold percentage for metric %q", m.Name),
			}
		}
		if seen[m.Name] {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: eval.metrics — duplicate metric name %q", m.Name),
				Fix:  fmt.Sprintf("Use unique names for each eval metric"),
			}
		}
		seen[m.Name] = true
	}
	return nil
}

func validateHostPort(af *Agentfile) error {
	if af.HostPort != 0 {
		if af.HostPort < 1 || af.HostPort > 65535 {
			return &output.UserError{
				Code: output.CodeInvalidAgentfile,
				What: fmt.Sprintf("Invalid field: host_port — %d is out of range", af.HostPort),
				Fix:  "Use a port between 1 and 65535",
			}
		}
	}
	// Check service host_ports for conflicts
	usedHostPorts := map[int]string{}
	names := make([]string, 0, len(af.Services))
	for name := range af.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		svc := af.Services[name]
		if svc.HostPort != 0 {
			if svc.HostPort < 1 || svc.HostPort > 65535 {
				return &output.UserError{
					Code: output.CodeInvalidAgentfile,
					What: fmt.Sprintf("Invalid field: services.%s.host_port — %d is out of range", name, svc.HostPort),
					Fix:  "Use a port between 1 and 65535",
				}
			}
			if other, ok := usedHostPorts[svc.HostPort]; ok {
				return &output.UserError{
					Code: output.CodeInvalidAgentfile,
					What: fmt.Sprintf("Invalid field: services.%s.host_port — %d already used by service %q", name, svc.HostPort, other),
					Fix:  "Each service must use a unique host_port",
				}
			}
			usedHostPorts[svc.HostPort] = name
		}
	}
	return nil
}
