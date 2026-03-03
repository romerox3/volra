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
	err := CopyGrafanaAssets(dir)
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
	err := CopyGrafanaAssets(dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/provisioning/datasources/datasource.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "http://prometheus:9090")
	assert.Contains(t, string(content), "isDefault: true")
	assert.Contains(t, string(content), "name: Prometheus")
}

func TestCopyGrafanaAssets_DashboardProviderContent(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "grafana/provisioning/dashboards/dashboards.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "/var/lib/grafana/dashboards")
	assert.Contains(t, string(content), "name: Volra")
}

func TestCopyGrafanaAssets_OverviewDashboard(t *testing.T) {
	dir := t.TempDir()
	err := CopyGrafanaAssets(dir)
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
	err := CopyGrafanaAssets(dir)
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
	err := CopyGrafanaAssets(dir)
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
	err := CopyGrafanaAssets(dir)
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
