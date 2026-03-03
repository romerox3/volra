package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// grafanaAssets maps embedded static file paths to their output paths under .volra/.
var grafanaAssets = []struct {
	src string
	dst string
}{
	{"static/datasource.yml", "grafana/provisioning/datasources/datasource.yml"},
	{"static/dashboards.yml", "grafana/provisioning/dashboards/dashboards.yml"},
	{"static/overview.json", "grafana/dashboards/overview.json"},
	{"static/detail.json", "grafana/dashboards/detail.json"},
}

// CopyGrafanaAssets copies all static Grafana files to the output directory.
func CopyGrafanaAssets(dir string) error {
	for _, asset := range grafanaAssets {
		data, err := staticFS.ReadFile(asset.src)
		if err != nil {
			return fmt.Errorf("reading %s: %w", asset.src, err)
		}

		outputPath := filepath.Join(dir, OutputDir, asset.dst)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", asset.dst, err)
		}

		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", asset.dst, err)
		}
	}
	return nil
}
