// Package agentfile handles Agentfile schema parsing and validation.
package agentfile

// HealthcheckConfig represents a Docker healthcheck for a service.
type HealthcheckConfig struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// TmpfsMount represents a tmpfs mount for read-only containers.
type TmpfsMount struct {
	Path string `yaml:"path"`
	Size string `yaml:"size,omitempty"`
}

// SecurityContext represents container security settings.
type SecurityContext struct {
	ReadOnly         bool         `yaml:"read_only,omitempty"`
	NoNewPrivileges  bool         `yaml:"no_new_privileges,omitempty"`
	DropCapabilities []string     `yaml:"drop_capabilities,omitempty"`
	Tmpfs            []TmpfsMount `yaml:"tmpfs,omitempty"`
}

// ResourceConfig represents container resource limits.
type ResourceConfig struct {
	MemLimit string `yaml:"mem_limit,omitempty"`
	CPUs     string `yaml:"cpus,omitempty"`
}

// BuildConfig represents build-time setup for ML and similar workloads.
type BuildConfig struct {
	SetupCommands []string `yaml:"setup_commands,omitempty"`
	CacheDirs     []string `yaml:"cache_dirs,omitempty"`
}

// ObservabilityConfig represents agent observability settings.
type ObservabilityConfig struct {
	Level       int `yaml:"level,omitempty"`        // 1 (probe only) or 2 (LLM metrics), default 1
	MetricsPort int `yaml:"metrics_port,omitempty"` // default 9101
}

// Service represents an infrastructure service declared in the Agentfile.
type Service struct {
	Image       string             `yaml:"image"`
	Port        int                `yaml:"port,omitempty"`
	HostPort    int                `yaml:"host_port,omitempty"`
	Volumes     []string           `yaml:"volumes,omitempty"`
	Env         []string           `yaml:"env,omitempty"`
	Healthcheck *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Resources   *ResourceConfig    `yaml:"resources,omitempty"`
}

// EvalMetric represents a single metric to evaluate.
type EvalMetric struct {
	Name      string `yaml:"name"`
	Query     string `yaml:"query"`
	Threshold int    `yaml:"threshold"`
	Unit      string `yaml:"unit,omitempty"`
}

// EvalConfig represents the evaluation configuration.
type EvalConfig struct {
	Metrics        []EvalMetric `yaml:"metrics"`
	BaselineWindow string       `yaml:"baseline_window,omitempty"` // default "1h"
}

// Agentfile is the central configuration model for a Volra agent project.
type Agentfile struct {
	Version       int                `yaml:"version"`
	Name          string             `yaml:"name"`
	Framework     Framework          `yaml:"framework"`
	Port          int                `yaml:"port"`
	HostPort      int                `yaml:"host_port,omitempty"`
	HealthPath    string             `yaml:"health_path"`
	HealthTimeout int                `yaml:"health_timeout,omitempty"`
	Volumes       []string           `yaml:"volumes,omitempty"`
	Env           []string           `yaml:"env,omitempty"`
	PackageManager PackageManager    `yaml:"package_manager,omitempty"`
	Dockerfile    DockerfileMode     `yaml:"dockerfile"`
	Services      map[string]Service `yaml:"services,omitempty"`
	Security      *SecurityContext   `yaml:"security,omitempty"`
	GPU           bool                `yaml:"gpu,omitempty"`
	Build         *BuildConfig        `yaml:"build,omitempty"`
	Observability *ObservabilityConfig `yaml:"observability,omitempty"`
	Eval          *EvalConfig          `yaml:"eval,omitempty"`
	Alerts        *AlertsConfig        `yaml:"alerts,omitempty"`
}

// AlertsConfig defines notification and alerting configuration.
type AlertsConfig struct {
	Channels []AlertChannel `yaml:"channels"`
	Rules    []AlertRule    `yaml:"rules,omitempty"`
}

// AlertChannel defines a notification channel.
type AlertChannel struct {
	Type        string `yaml:"type"`                    // slack, email, webhook
	WebhookEnv  string `yaml:"webhook_url_env,omitempty"` // env var name for Slack webhook
	To          string `yaml:"to,omitempty"`             // email recipient
	SMTPEnv     string `yaml:"smtp_env,omitempty"`       // env var name for SMTP config
	URL         string `yaml:"url,omitempty"`            // webhook URL
}

// AlertRule defines a custom Prometheus alerting rule.
type AlertRule struct {
	Name     string `yaml:"name"`
	Expr     string `yaml:"expr"`
	For      string `yaml:"for,omitempty"`
	Severity string `yaml:"severity,omitempty"`
}
