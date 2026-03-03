package status

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/antonioromero/volra/internal/output"
	"github.com/antonioromero/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeAgentfile(t *testing.T, dir string, port int) {
	t.Helper()
	content := fmt.Sprintf(`version: 1
name: test-agent
framework: generic
port: %d
health_path: /health
dockerfile: auto
`, port)
	err := os.WriteFile(filepath.Join(dir, "Agentfile"), []byte(content), 0644)
	require.NoError(t, err)
}

func composePsJSON(name string, states map[string]string) string {
	var lines []string
	for svc, state := range states {
		lines = append(lines, fmt.Sprintf(`{"Name":"%s-%s","State":"%s"}`, name, svc, state))
	}
	return strings.Join(lines, "\n")
}

func extractPort(t *testing.T, url string) int {
	t.Helper()
	parts := strings.Split(url, ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	require.NoError(t, err)
	return port
}

func TestRun_NoAgentfile(t *testing.T) {
	dir := t.TempDir()
	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{Responses: map[string]testutil.MockResponse{}}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeNoDeployment, ue.Code)
}

func TestRun_DockerNotRunning(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir, 8000)

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info": {Output: "", Err: errors.New("docker not running")},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeStatusDockerNotRunning, ue.Code)
}

func TestRun_NoContainersRunning(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir, 8000)
	composePath := filepath.Join(dir, outputDir, "docker-compose.yml")

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info": {Output: "Docker info", Err: nil},
			fmt.Sprintf("compose -f %s -p test-agent ps --format json", composePath): {
				Output: "", Err: errors.New("no containers"),
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeNoDeployment, ue.Code)
}

func TestRun_AllRunning_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	dir := t.TempDir()
	writeAgentfile(t, dir, port)
	composePath := filepath.Join(dir, outputDir, "docker-compose.yml")

	psOutput := composePsJSON("test-agent", map[string]string{
		"agent":      "running",
		"prometheus": "running",
		"grafana":    "running",
	})

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info": {Output: "Docker info", Err: nil},
			fmt.Sprintf("compose -f %s -p test-agent ps --format json", composePath): {
				Output: psOutput, Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	results := strings.Join(p.ResultCalls, "\n")
	assert.Contains(t, results, "healthy")
	assert.Contains(t, results, fmt.Sprintf("port %d", port))
	assert.Contains(t, results, "Prometheus")
	assert.Contains(t, results, "Grafana")
}

func TestRun_AgentUnhealthy(t *testing.T) {
	// Server returns 503
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	dir := t.TempDir()
	writeAgentfile(t, dir, port)
	composePath := filepath.Join(dir, outputDir, "docker-compose.yml")

	psOutput := composePsJSON("test-agent", map[string]string{
		"agent":      "running",
		"prometheus": "running",
		"grafana":    "running",
	})

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info": {Output: "Docker info", Err: nil},
			fmt.Sprintf("compose -f %s -p test-agent ps --format json", composePath): {
				Output: psOutput, Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	results := strings.Join(p.ResultCalls, "\n")
	assert.Contains(t, results, "unhealthy")

	// Should have a warning about unhealthy
	require.NotEmpty(t, p.WarnCalls)
	assert.Contains(t, p.WarnCalls[0].What, "health check failed")
}

func TestRun_AllStopped_DaemonRestart(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir, 8000)
	composePath := filepath.Join(dir, outputDir, "docker-compose.yml")

	psOutput := composePsJSON("test-agent", map[string]string{
		"agent":      "exited",
		"prometheus": "exited",
		"grafana":    "exited",
	})

	p := &testutil.MockPresenter{}
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info": {Output: "Docker info", Err: nil},
			fmt.Sprintf("compose -f %s -p test-agent ps --format json", composePath): {
				Output: psOutput, Err: nil,
			},
		},
	}

	err := Run(context.Background(), dir, p, mock)
	require.NoError(t, err)

	// Should warn about daemon restart
	require.NotEmpty(t, p.WarnCalls)
	found := false
	for _, w := range p.WarnCalls {
		if strings.Contains(w.What, "stopped") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected warning about stopped containers")
}

func TestAllStopped_AllExited(t *testing.T) {
	services := []ServiceState{
		{Name: "test-agent", State: "exited"},
		{Name: "test-prometheus", State: "exited"},
	}
	assert.True(t, allStopped(services))
}

func TestAllStopped_SomeRunning(t *testing.T) {
	services := []ServiceState{
		{Name: "test-agent", State: "running"},
		{Name: "test-prometheus", State: "exited"},
	}
	assert.False(t, allStopped(services))
}

func TestAllStopped_Empty(t *testing.T) {
	assert.False(t, allStopped(nil))
}

func TestFindState_Found(t *testing.T) {
	services := []ServiceState{
		{Name: "my-agent-agent", State: "running"},
		{Name: "my-agent-prometheus", State: "running"},
	}
	assert.Equal(t, "running", findState(services, "agent"))
	assert.Equal(t, "running", findState(services, "prometheus"))
}

func TestFindState_NotFound(t *testing.T) {
	services := []ServiceState{
		{Name: "my-agent-agent", State: "running"},
	}
	assert.Equal(t, "not found", findState(services, "grafana"))
}

func TestProbeHealth_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	port := extractPort(t, srv.URL)
	assert.True(t, probeHealth(context.Background(), port, "/"))
}

func TestProbeHealth_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	port := extractPort(t, srv.URL)
	assert.False(t, probeHealth(context.Background(), port, "/"))
}

func TestProbeHealth_ConnectionRefused(t *testing.T) {
	assert.False(t, probeHealth(context.Background(), 19999, "/health"))
}
