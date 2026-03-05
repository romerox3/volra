package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyGrafanaAssets_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	expected := []string{
		filepath.Join(dir, OutputDir, "grafana/provisioning/datasources/datasource.yml"),
		filepath.Join(dir, OutputDir, "grafana/provisioning/dashboards/dashboards.yml"),
		filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"),
		filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"),
	}
	for _, path := range expected {
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected file %s to exist", path)
	}
}

func TestCopyGrafanaAssets_DatasourceContent(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/provisioning/datasources/datasource.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "http://prometheus:9090")
	assert.Contains(t, string(content), "isDefault: true")
	assert.Contains(t, string(content), "name: Prometheus")
}

func TestCopyGrafanaAssets_DashboardProviderContent(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/provisioning/dashboards/dashboards.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "/var/lib/grafana/dashboards")
	assert.Contains(t, string(content), "name: Volra")
}

func TestCopyGrafanaAssets_OverviewDashboard(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Agent Overview")
	assert.Contains(t, string(content), "volra-overview")
	assert.Contains(t, string(content), "Agent Status (Probe)")
	assert.Contains(t, string(content), "agent-health")
}

func TestCopyGrafanaAssets_DetailDashboard(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Agent Detail")
	assert.Contains(t, string(content), "volra-detail")
	assert.Contains(t, string(content), "Probe Latency Over Time")
	assert.Contains(t, string(content), "Health Status Timeline")
	assert.Contains(t, string(content), "Uptime (Probe, 24h)")
	assert.Contains(t, string(content), "Uptime (Probe, 7d)")
}

func TestCopyGrafanaAssets_ProbeLabeling(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	// FR37: All metrics clearly labeled as probe-based
	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	assert.Contains(t, string(overview), "Probe")

	detail, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	assert.Contains(t, string(detail), "Probe")
}

func TestCopyGrafanaAssets_CreatesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	dirs := []string{
		filepath.Join(dir, OutputDir, "grafana/provisioning/datasources"),
		filepath.Join(dir, OutputDir, "grafana/provisioning/dashboards"),
		filepath.Join(dir, OutputDir, "grafana/dashboards"),
	}
	for _, d := range dirs {
		info, err := os.Stat(d)
		require.NoError(t, err, "expected directory %s to exist", d)
		assert.True(t, info.IsDir())
	}
}

// --- HasMetrics dashboard tests ---

func TestCopyGrafanaAssets_NoMetrics_ProbeOnly(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	// Should NOT contain custom metrics panels
	assert.NotContains(t, string(overview), "Request Rate")
	assert.NotContains(t, string(overview), "Active Requests")
	assert.NotContains(t, string(overview), "agent-metrics")
}

func TestCopyGrafanaAssets_WithMetrics_OverviewHasCustomPanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	// Should contain probe panels AND custom metrics panels
	assert.Contains(t, string(overview), "Agent Status (Probe)")
	assert.Contains(t, string(overview), "Request Rate")
	assert.Contains(t, string(overview), "Active Requests")
	assert.Contains(t, string(overview), "agent-metrics")
	assert.Contains(t, string(overview), "Application-reported metric")
}

func TestCopyGrafanaAssets_WithMetrics_DetailHasCustomPanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	detail, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	// Should contain probe panels AND custom metrics panels
	assert.Contains(t, string(detail), "Probe Latency Over Time")
	assert.Contains(t, string(detail), "Request Rate Over Time")
	assert.Contains(t, string(detail), "Request Duration (P95)")
	assert.Contains(t, string(detail), "Custom Agent Metrics")
	assert.Contains(t, string(detail), "agent-metrics")
	assert.Contains(t, string(detail), "Application-reported metric")
}

func TestCopyGrafanaAssets_WithMetrics_StillHasProbePanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	// Probe panels must still be present
	assert.Contains(t, string(overview), "agent-health")
	assert.Contains(t, string(overview), "probe_success")
}

func TestSelectDashboardSrc(t *testing.T) {
	assert.Equal(t, "static/overview.json", selectDashboardSrc("overview", false))
	assert.Equal(t, "static/overview_metrics.json", selectDashboardSrc("overview", true))
	assert.Equal(t, "static/detail.json", selectDashboardSrc("detail", false))
	assert.Equal(t, "static/detail_metrics.json", selectDashboardSrc("detail", true))
}

// --- LLM Token Tracking Panel tests (Story 7.2) ---

func TestCopyGrafanaAssets_WithMetrics_OverviewHasLLMPanel(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	assert.Contains(t, string(overview), "LLM Token Rate")
	assert.Contains(t, string(overview), "llm_tokens_total")
	assert.Contains(t, string(overview), "Volra LLM Metrics Convention")
}

func TestCopyGrafanaAssets_WithMetrics_DetailHasLLMPanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	detail, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	// Token Consumption panel
	assert.Contains(t, string(detail), "Token Consumption Over Time")
	assert.Contains(t, string(detail), "llm_tokens_total")
	// Cost Trending panel
	assert.Contains(t, string(detail), "LLM Cost Trending")
	assert.Contains(t, string(detail), "llm_request_cost_dollars_total")
	// Per-Model Breakdown panel
	assert.Contains(t, string(detail), "Per-Model Request Breakdown")
	assert.Contains(t, string(detail), "llm_model_requests_total")
	// All panels reference Volra LLM Metrics Convention
	assert.Contains(t, string(detail), "Volra LLM Metrics Convention")
}

func TestCopyGrafanaAssets_WithMetrics_LLMPanelsUseAgentMetricsJob(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, true, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	// LLM panel uses same agent-metrics job
	assert.Contains(t, string(overview), "agent-metrics")

	detail, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	assert.Contains(t, string(detail), "agent-metrics")
}

func TestCopyGrafanaAssets_NoMetrics_NoLLMPanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(overview), "llm_tokens_total")
	assert.NotContains(t, string(overview), "LLM Token Rate")

	detail, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(detail), "llm_tokens_total")
	assert.NotContains(t, string(detail), "llm_request_cost_dollars_total")
	assert.NotContains(t, string(detail), "llm_model_requests_total")
}

// --- Level 2 LLM Observability Dashboard tests (Story 13.4) ---

func TestCopyGrafanaAssets_NoLevel2_NoDashboard(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, false)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	assert.True(t, os.IsNotExist(err), "level2.json should not exist when hasLevel2=false")
}

func TestCopyGrafanaAssets_WithLevel2_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, true)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	assert.NoError(t, err, "level2.json should exist when hasLevel2=true")
}

func TestCopyGrafanaAssets_Level2_DashboardContent(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, true)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	require.NoError(t, err)
	s := string(content)

	// Dashboard metadata
	assert.Contains(t, s, "volra-level2")
	assert.Contains(t, s, "Agent LLM Observability")

	// All 5 core volra-observe metrics present
	assert.Contains(t, s, "volra_llm_tokens_total")
	assert.Contains(t, s, "volra_llm_cost_dollars_total")
	assert.Contains(t, s, "volra_llm_request_duration_seconds_bucket")
	assert.Contains(t, s, "volra_llm_errors_total")
	assert.Contains(t, s, "volra_tool_calls_total")

	// Uses agent-level2 job (from prometheus.yml.tmpl)
	assert.Contains(t, s, "agent-level2")
}

func TestCopyGrafanaAssets_Level2_HasExpectedPanels(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, true)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	require.NoError(t, err)
	s := string(content)

	// 7 key panels
	assert.Contains(t, s, "Daily LLM Cost")
	assert.Contains(t, s, "Total Tokens (24h)")
	assert.Contains(t, s, "LLM Error Rate")
	assert.Contains(t, s, "Token Rate by Type")
	assert.Contains(t, s, "LLM Cost Trending")
	assert.Contains(t, s, "LLM Request Latency")
	assert.Contains(t, s, "LLM Errors by Type")
	assert.Contains(t, s, "Tool Call Frequency")
	assert.Contains(t, s, "Cost per Model")
}

func TestCopyGrafanaAssets_Level2_LatencyPercentiles(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir, false, true)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	require.NoError(t, err)
	s := string(content)

	// P50, P95, P99 latency quantiles
	assert.Contains(t, s, "histogram_quantile(0.50")
	assert.Contains(t, s, "histogram_quantile(0.95")
	assert.Contains(t, s, "histogram_quantile(0.99")
}

func TestCopyGrafanaAssets_Level2_WithMetrics_BothDashboards(t *testing.T) {
	dir := t.TempDir()
	// Both hasMetrics AND hasLevel2 enabled
	err := CopyGrafanaAssets(dir, true, true)
	require.NoError(t, err)

	// Standard metrics dashboards should use _metrics variants
	overview, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"))
	require.NoError(t, err)
	assert.Contains(t, string(overview), "Request Rate")

	// Level 2 dashboard should also exist
	level2, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/dashboards/level2.json"))
	require.NoError(t, err)
	assert.Contains(t, string(level2), "volra-level2")
}
