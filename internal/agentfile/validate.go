package agentfile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antonioromero/volra/internal/output"
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
	if err := validateEnv(af.Env); err != nil {
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
