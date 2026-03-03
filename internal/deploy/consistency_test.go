package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedTemplates_Exist(t *testing.T) {
	templates := []string{
		"templates/Dockerfile.tmpl",
		"templates/docker-compose.yml.tmpl",
		"templates/prometheus.yml.tmpl",
	}
	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			_, err := templateFS.ReadFile(name)
			require.NoError(t, err, "embedded template %s must exist", name)
		})
	}
}

func TestEmbeddedStatic_Exist(t *testing.T) {
	statics := []string{
		"static/alert_rules.yml",
		"static/datasource.yml",
		"static/dashboards.yml",
		"static/overview.json",
		"static/detail.json",
	}
	for _, name := range statics {
		t.Run(name, func(t *testing.T) {
			_, err := staticFS.ReadFile(name)
			require.NoError(t, err, "embedded static file %s must exist", name)
		})
	}
}
