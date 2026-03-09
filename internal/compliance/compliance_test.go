package compliance

import (
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_BasicAgent(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:       "test-agent",
		Framework:  "generic",
		Version:    1,
		Port:       8000,
		HealthPath: "/health",
	}

	out, err := Generate(af)
	require.NoError(t, err)

	assert.Contains(t, out, "test-agent")
	assert.Contains(t, out, "generic")
	assert.Contains(t, out, "/health")
	assert.Contains(t, out, "EU AI Act Compliance")
}

func TestGenerate_WithObservability(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:       "obs-agent",
		Framework:  "langchain",
		Version:    1,
		Port:       8000,
		HealthPath: "/health",
		Observability: &agentfile.ObservabilityConfig{
			Level:       2,
			MetricsPort: 9101,
		},
	}

	out, err := Generate(af)
	require.NoError(t, err)

	assert.Contains(t, out, "Observability Level")
	assert.Contains(t, out, "LLM execution traces")
}

func TestGenerate_WithAlerting(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:       "alert-agent",
		Framework:  "generic",
		Version:    1,
		Port:       8000,
		HealthPath: "/health",
		Alerts: &agentfile.AlertsConfig{
			Channels: []agentfile.AlertChannel{
				{Type: "slack"},
				{Type: "email", To: "ops@example.com"},
			},
			Rules: []agentfile.AlertRule{
				{Name: "high_cpu", Expr: "cpu > 90", For: "5m", Severity: "warning"},
			},
		},
	}

	out, err := Generate(af)
	require.NoError(t, err)

	assert.Contains(t, out, "Alertmanager")
	assert.Contains(t, out, "slack, email")
	assert.Contains(t, out, "high_cpu")
	// Default rules should also appear.
	assert.Contains(t, out, "agent_down")
}

func TestGenerate_NilAgentfile(t *testing.T) {
	_, err := Generate(nil)
	assert.Error(t, err)
}

func TestGenerate_NoAlertingShowsPrompt(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:       "basic",
		Framework:  "generic",
		Version:    1,
		Port:       8000,
		HealthPath: "/health",
	}

	out, err := Generate(af)
	require.NoError(t, err)

	assert.Contains(t, out, "No automated alerting configured")
}
