package alerting

import (
	"strings"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRules_ReturnsThreeRules(t *testing.T) {
	rules := DefaultRules()
	require.Len(t, rules, 3)

	assert.Equal(t, "agent_down", rules[0].Name)
	assert.Equal(t, "critical", rules[0].Severity)

	assert.Equal(t, "high_latency_p95", rules[1].Name)
	assert.Equal(t, "warning", rules[1].Severity)

	assert.Equal(t, "high_error_rate", rules[2].Name)
	assert.Equal(t, "warning", rules[2].Severity)
}

func TestRenderAlertmanagerConfig_SlackChannel(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Channels: []agentfile.AlertChannel{
			{Type: "slack", WebhookEnv: "SLACK_WEBHOOK_URL"},
		},
	}

	out, err := RenderAlertmanagerConfig(cfg)
	require.NoError(t, err)

	assert.Contains(t, out, "volra-default")
	assert.Contains(t, out, "slack_configs")
	assert.Contains(t, out, "slack-webhook-url")
}

func TestRenderAlertmanagerConfig_EmailChannel(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Channels: []agentfile.AlertChannel{
			{Type: "email", To: "ops@example.com"},
		},
	}

	out, err := RenderAlertmanagerConfig(cfg)
	require.NoError(t, err)

	assert.Contains(t, out, "email_configs")
	assert.Contains(t, out, "ops@example.com")
}

func TestRenderAlertmanagerConfig_WebhookChannel(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Channels: []agentfile.AlertChannel{
			{Type: "webhook", URL: "https://hooks.example.com/alert"},
		},
	}

	out, err := RenderAlertmanagerConfig(cfg)
	require.NoError(t, err)

	assert.Contains(t, out, "webhook_configs")
	assert.Contains(t, out, "https://hooks.example.com/alert")
}

func TestRenderAlertmanagerConfig_NilReturnsError(t *testing.T) {
	_, err := RenderAlertmanagerConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no alert channels configured")
}

func TestRenderAlertmanagerConfig_EmptyChannelsReturnsError(t *testing.T) {
	cfg := &agentfile.AlertsConfig{Channels: []agentfile.AlertChannel{}}
	_, err := RenderAlertmanagerConfig(cfg)
	assert.Error(t, err)
}

func TestRenderAlertRules_DefaultsOnly(t *testing.T) {
	out, err := RenderAlertRules(nil)
	require.NoError(t, err)

	assert.Contains(t, out, "volra-agent-alerts")
	assert.Contains(t, out, "agent_down")
	assert.Contains(t, out, "high_latency_p95")
	assert.Contains(t, out, "high_error_rate")
}

func TestRenderAlertRules_MergesCustomRules(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Rules: []agentfile.AlertRule{
			{Name: "custom_alert", Expr: "some_metric > 100"},
		},
	}

	out, err := RenderAlertRules(cfg)
	require.NoError(t, err)

	// Should have defaults + custom.
	assert.Contains(t, out, "agent_down")
	assert.Contains(t, out, "custom_alert")
	assert.Contains(t, out, "some_metric > 100")
}

func TestRenderAlertRules_SetsDefaultForAndSeverity(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Rules: []agentfile.AlertRule{
			{Name: "bare_rule", Expr: "up == 0"},
		},
	}

	out, err := RenderAlertRules(cfg)
	require.NoError(t, err)

	// The custom rule should get default for: 5m and severity: warning.
	lines := strings.Split(out, "\n")
	foundBareRule := false
	for i, line := range lines {
		if strings.Contains(line, "bare_rule") {
			foundBareRule = true
			// Check subsequent lines for defaults.
			remaining := strings.Join(lines[i:], "\n")
			assert.Contains(t, remaining, "for: 5m")
			assert.Contains(t, remaining, "severity: warning")
			break
		}
	}
	assert.True(t, foundBareRule, "bare_rule not found in output")
}

func TestComposeService_ReturnsValidMap(t *testing.T) {
	svc := ComposeService()

	assert.Equal(t, "prom/alertmanager:v0.27.0", svc["image"])
	assert.Equal(t, "unless-stopped", svc["restart"])

	volumes, ok := svc["volumes"].([]string)
	require.True(t, ok)
	assert.Len(t, volumes, 1)

	ports, ok := svc["ports"].([]string)
	require.True(t, ok)
	assert.Contains(t, ports, "9093:9093")
}

func TestRenderAlertmanagerConfig_MultipleChannels(t *testing.T) {
	cfg := &agentfile.AlertsConfig{
		Channels: []agentfile.AlertChannel{
			{Type: "slack", WebhookEnv: "SLACK_URL"},
			{Type: "email", To: "team@example.com"},
			{Type: "webhook", URL: "https://hooks.example.com"},
		},
	}

	out, err := RenderAlertmanagerConfig(cfg)
	require.NoError(t, err)

	assert.Contains(t, out, "slack_configs")
	assert.Contains(t, out, "email_configs")
	assert.Contains(t, out, "webhook_configs")
}
