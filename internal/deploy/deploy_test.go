package deploy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeAgentfile(t *testing.T, dir string, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "Agentfile"), []byte(content), 0644)
	require.NoError(t, err)
}

func writeRequirements(t *testing.T, dir string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("fastapi\n"), 0644)
	require.NoError(t, err)
}

func writeEntryPoint(t *testing.T, dir string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')\n"), 0644)
	require.NoError(t, err)
}

func writeDotEnv(t *testing.T, dir string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=test\n"), 0644)
	require.NoError(t, err)
}

const minimalAgentfile = `version: 1
name: test-agent
framework: generic
port: %d
health_path: /health
dockerfile: auto
`

const agentfileWithEnv = `version: 1
name: test-agent
framework: generic
port: %d
health_path: /health
dockerfile: auto
env:
  - API_KEY
`

func TestRun_MissingAgentfile(t *testing.T) {
	dir := t.TempDir()
	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{Responses: map[string]testutil.MockResponse{}}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opening Agentfile")
}

func TestRun_MissingEnvFile(t *testing.T) {
	dir := t.TempDir()
	// Start a test server so health check could work (though we shouldn't reach it)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	writeAgentfile(t, dir, fmt.Sprintf(agentfileWithEnv, port))
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)
	// Intentionally NOT creating .env

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{Responses: map[string]testutil.MockResponse{}}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeEnvNotFound, ue.Code)
	assert.Contains(t, ue.Fix, ".env.example")
}

func TestRun_GeneratesArtifacts(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	writeAgentfile(t, dir, fmt.Sprintf(minimalAgentfile, port))
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)

	p := &testutil.MockPresenter{}
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s -p test-agent up -d --build", composePath): {
				Output: "done\n", Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	// Check artifacts were generated
	artifacts := []string{
		filepath.Join(dir, OutputDir, "Dockerfile"),
		filepath.Join(dir, OutputDir, "docker-compose.yml"),
		filepath.Join(dir, OutputDir, "prometheus.yml"),
		filepath.Join(dir, OutputDir, "alert_rules.yml"),
		filepath.Join(dir, OutputDir, "grafana/dashboards/overview.json"),
		filepath.Join(dir, OutputDir, "grafana/dashboards/detail.json"),
		filepath.Join(dir, OutputDir, "grafana/provisioning/datasources/datasource.yml"),
		filepath.Join(dir, OutputDir, "grafana/provisioning/dashboards/dashboards.yml"),
		filepath.Join(dir, OutputDir, "agent-card.json"),
	}
	for _, path := range artifacts {
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected artifact %s to exist", path)
	}
}

func TestGenerateA2ACard(t *testing.T) {
	dir := t.TempDir()

	af := &agentfile.Agentfile{
		Name:      "test-agent",
		Framework: agentfile.FrameworkGeneric,
		Port:      8000,
	}
	tc := &TemplateContext{
		AgentHostPort: 8000,
	}

	err := GenerateA2ACard(af, tc, dir)
	require.NoError(t, err)

	cardPath := filepath.Join(dir, OutputDir, "agent-card.json")
	data, err := os.ReadFile(cardPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-agent")
	assert.Contains(t, string(data), "http://localhost:8000")
	assert.Contains(t, string(data), "generic")
}

func TestRun_OutputSummary(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	writeAgentfile(t, dir, fmt.Sprintf(minimalAgentfile, port))
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)

	p := &testutil.MockPresenter{}
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s -p test-agent up -d --build", composePath): {
				Output: "done\n", Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	// Check summary output
	results := joinStrings(p.ResultCalls)
	assert.Contains(t, results, fmt.Sprintf("http://localhost:%d", port))
	assert.Contains(t, results, "http://localhost:3001")
	assert.Contains(t, results, "http://localhost:9090")
	assert.Contains(t, results, "down")
}

func TestRun_DockerComposeError(t *testing.T) {
	dir := t.TempDir()

	writeAgentfile(t, dir, fmt.Sprintf(minimalAgentfile, 8000))
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)

	p := &testutil.MockPresenter{}
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s -p test-agent up -d --build", composePath): {
				Output: "Cannot connect to the Docker daemon",
				Err:    errors.New("exit status 1"),
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeDeployDockerNotRunning, ue.Code)
}

func TestRun_SkipsDockerfileForCustomMode(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	customAF := fmt.Sprintf(`version: 1
name: test-agent
framework: generic
port: %d
health_path: /health
dockerfile: custom
`, port)
	writeAgentfile(t, dir, customAF)
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)

	p := &testutil.MockPresenter{}
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s -p test-agent up -d --build", composePath): {
				Output: "done\n", Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	// Dockerfile should NOT be generated for custom mode
	_, err = os.Stat(filepath.Join(dir, OutputDir, "Dockerfile"))
	assert.True(t, os.IsNotExist(err), "Dockerfile should not exist for custom mode")
}

func TestRun_WithEnvFile(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	writeAgentfile(t, dir, fmt.Sprintf(agentfileWithEnv, port))
	writeRequirements(t, dir)
	writeEntryPoint(t, dir)
	writeDotEnv(t, dir)

	p := &testutil.MockPresenter{}
	composePath := filepath.Join(dir, OutputDir, "docker-compose.yml")
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s -p test-agent up -d --build", composePath): {
				Output: "done\n", Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)
}

func joinStrings(ss []string) string {
	result := ""
	for _, s := range ss {
		result += s + "\n"
	}
	return result
}
