// Package agentfile handles Agentfile schema parsing and validation.
package agentfile

// Agentfile is the central configuration model for a Volra agent project.
type Agentfile struct {
	Version    int            `yaml:"version"`
	Name       string         `yaml:"name"`
	Framework  Framework      `yaml:"framework"`
	Port       int            `yaml:"port"`
	HealthPath string         `yaml:"health_path"`
	Env        []string       `yaml:"env,omitempty"`
	Dockerfile DockerfileMode `yaml:"dockerfile"`
}
