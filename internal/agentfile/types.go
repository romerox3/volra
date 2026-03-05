package agentfile

import (
	"fmt"

	"github.com/romerox3/volra/internal/output"
	"gopkg.in/yaml.v3"
)

// Framework represents the agent framework type.
type Framework string

const (
	// FrameworkGeneric is the default framework for generic Python agents.
	FrameworkGeneric Framework = "generic"
	// FrameworkLangGraph is the framework for LangGraph-based agents.
	FrameworkLangGraph Framework = "langgraph"
)

// UnmarshalYAML validates the framework value during YAML parsing.
func (f *Framework) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch Framework(s) {
	case FrameworkGeneric, FrameworkLangGraph:
		*f = Framework(s)
		return nil
	default:
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: framework — %q is not a valid framework", s),
			Fix:  "Use 'generic' or 'langgraph'",
		}
	}
}

// PackageManager represents the Python package manager used by the project.
type PackageManager string

const (
	// PackageManagerPip is the default package manager (pip + requirements.txt or pyproject.toml).
	PackageManagerPip PackageManager = "pip"
	// PackageManagerPoetry uses Poetry for dependency management.
	PackageManagerPoetry PackageManager = "poetry"
	// PackageManagerUV uses uv for dependency management.
	PackageManagerUV PackageManager = "uv"
	// PackageManagerPipenv uses Pipenv for dependency management.
	PackageManagerPipenv PackageManager = "pipenv"
)

// UnmarshalYAML validates the package manager value during YAML parsing.
func (pm *PackageManager) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch PackageManager(s) {
	case PackageManagerPip, PackageManagerPoetry, PackageManagerUV, PackageManagerPipenv:
		*pm = PackageManager(s)
		return nil
	default:
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: package_manager — %q is not a valid package manager", s),
			Fix:  "Use 'pip', 'poetry', 'uv', or 'pipenv'",
		}
	}
}

// DockerfileMode represents how the Dockerfile is managed.
type DockerfileMode string

const (
	// DockerfileModeAuto means Volra generates the Dockerfile.
	DockerfileModeAuto DockerfileMode = "auto"
	// DockerfileModeCustom means the user provides a Dockerfile.
	DockerfileModeCustom DockerfileMode = "custom"
)

// UnmarshalYAML validates the dockerfile mode during YAML parsing.
func (d *DockerfileMode) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch DockerfileMode(s) {
	case DockerfileModeAuto, DockerfileModeCustom:
		*d = DockerfileMode(s)
		return nil
	default:
		return &output.UserError{
			Code: output.CodeInvalidAgentfile,
			What: fmt.Sprintf("Invalid field: dockerfile — %q is not a valid mode", s),
			Fix:  "Use 'auto' or 'custom'",
		}
	}
}
