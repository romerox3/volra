package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// grafanaBaseAssets are always copied regardless of HasMetrics.
var grafanaBaseAssets = []struct {
	src string
	dst string
}{
	{"static/datasource.yml", "grafana/provisioning/datasources/datasource.yml"},
	{"static/dashboards.yml", "grafana/provisioning/dashboards/dashboards.yml"},
}

// selectDashboardSrc returns the appropriate dashboard source path based on HasMetrics.
func selectDashboardSrc(name string, hasMetrics bool) string {
	if hasMetrics {
		return "static/" + name + "_metrics.json"
	}
	return "static/" + name + ".json"
}

// CopyGrafanaAssets copies all static Grafana files to the output directory.
// Dashboard selection depends on HasMetrics: if the agent exposes prometheus_client
// metrics, the enhanced dashboard variants (with custom metrics panels) are used.
// When hasLevel2 is true, the Level 2 LLM observability dashboard is also included.
func CopyGrafanaAssets(dir string, hasMetrics, hasLevel2 bool) error {
	// Copy base provisioning assets
	for _, asset := range grafanaBaseAssets {
		if err := copyStaticAsset(asset.src, asset.dst, dir); err != nil {
			return err
		}
	}

	// Copy dashboard variants based on HasMetrics
	dashboards := []struct {
		name string
		dst  string
	}{
		{"overview", "grafana/dashboards/overview.json"},
		{"detail", "grafana/dashboards/detail.json"},
	}

	for _, db := range dashboards {
		src := selectDashboardSrc(db.name, hasMetrics)
		if err := copyStaticAsset(src, db.dst, dir); err != nil {
			return err
		}
	}

	// Copy Level 2 LLM observability dashboard when enabled
	if hasLevel2 {
		if err := copyStaticAsset("static/level2.json", "grafana/dashboards/level2.json", dir); err != nil {
			return err
		}
	}

	return nil
}

// copyStaticAsset reads an embedded static file and writes it to the output directory.
func copyStaticAsset(src, dst, dir string) error {
	data, err := staticFS.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}

	outputPath := filepath.Join(dir, OutputDir, dst)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dst, err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	return nil
}
