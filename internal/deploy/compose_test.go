package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antonioromero/volra/internal/agentfile"
	"github.com/antonioromero/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCompose_Minimal(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "my-agent", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	testutil.AssertGolden(t, got, filepath.Join("testdata", "golden", "compose_minimal.golden"))
}

func TestRenderCompose_WithEnv(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "my-agent", Framework: agentfile.FrameworkGeneric,
			Port: 9000, HealthPath: "/healthz",
			Env:        []string{"API_KEY", "DB_URL"},
			Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.12",
		EntryPoint:      "app.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	testutil.AssertGolden(t, got, filepath.Join("testdata", "golden", "compose_with_env.golden"))
}

func TestRenderCompose_CustomDockerfile(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "my-agent", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeCustom,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "dockerfile: Dockerfile")
	assert.NotContains(t, got, ".volra/Dockerfile")
}

func TestRenderCompose_ThreeServices(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "agent:")
	assert.Contains(t, got, "prometheus:")
	assert.Contains(t, got, "grafana:")
}

func TestRenderCompose_ProjectName(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "cool-agent", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "name: cool-agent")
	assert.Contains(t, got, "cool-agent-agent")
	assert.Contains(t, got, "cool-agent-prometheus")
	assert.Contains(t, got, "cool-agent-grafana")
}

func TestRenderCompose_PortMapping(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 3000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, `"3000:3000"`)
	assert.Contains(t, got, `"9090:9090"`)
	assert.Contains(t, got, `"3001:3000"`)
}

func TestRenderCompose_PrometheusVolumes(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "prometheus.yml:/etc/prometheus/prometheus.yml:ro")
	assert.Contains(t, got, "alert_rules.yml:/etc/prometheus/alert_rules.yml:ro")
	assert.Contains(t, got, "prometheus-data:/prometheus")
}

func TestRenderCompose_GrafanaConfig(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "GF_AUTH_ANONYMOUS_ENABLED=true")
	assert.Contains(t, got, "GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer")
	assert.Contains(t, got, "GF_AUTH_DISABLE_LOGIN_FORM=true")
	assert.Contains(t, got, "grafana/dashboards:/var/lib/grafana/dashboards:ro")
	assert.Contains(t, got, "grafana/provisioning:/etc/grafana/provisioning:ro")
}

func TestGenerateCompose_WritesFile(t *testing.T) {
	dir := t.TempDir()
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	err := GenerateCompose(tc, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Generated by Volra")
	assert.Contains(t, string(content), "name: test")
}

func TestRenderCompose_NoEnvFile(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.NotContains(t, got, "env_file")
}

func TestRenderCompose_Network(t *testing.T) {
	tc := &TemplateContext{
		Agentfile: agentfile.Agentfile{
			Version: 1, Name: "test", Framework: agentfile.FrameworkGeneric,
			Port: 8000, HealthPath: "/health", Dockerfile: agentfile.DockerfileModeAuto,
		},
		PythonVersion:   "3.11",
		EntryPoint:      "main.py",
		HasRequirements: true,
	}
	got, err := RenderCompose(tc)
	require.NoError(t, err)
	assert.Contains(t, got, "networks:")
	assert.Contains(t, got, "volra:")
	assert.Contains(t, got, "driver: bridge")
}
