package hub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHub(t *testing.T) (string, string) {
	t.Helper()

	// Override registry path.
	regDir := t.TempDir()
	regPath := filepath.Join(regDir, "agents.json")
	origRegPath := registry.PathFunc
	registry.PathFunc = func() (string, error) { return regPath, nil }
	t.Cleanup(func() { registry.PathFunc = origRegPath })

	// Override hub dir.
	hubDir := filepath.Join(t.TempDir(), "hub")
	origHubDir := HubDirFunc
	HubDirFunc = func() (string, error) { return hubDir, nil }
	t.Cleanup(func() { HubDirFunc = origHubDir })

	return regPath, hubDir
}

func TestStart_NoAgentsRegistered(t *testing.T) {
	setupTestHub(t)

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{}

	err := Start(context.Background(), mp, mr)

	require.Error(t, err)
	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeNoAgentsRegistered, ue.Code)
}

func TestStart_GeneratesArtifacts(t *testing.T) {
	_, hubDir := setupTestHub(t)

	// Register two agents with real directories.
	dirA := t.TempDir()
	dirB := t.TempDir()
	require.NoError(t, registry.Register("agent-a", dirA, 9090, 8000))
	require.NoError(t, registry.Register("agent-b", dirB, 9091, 8001))

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"compose -f " + filepath.Join(hubDir, "docker-compose.yml") + " -p volra-hub ps --quiet": {Output: "", Err: os.ErrNotExist},
			"compose -f " + filepath.Join(hubDir, "docker-compose.yml") + " -p volra-hub up -d":     {Output: "started", Err: nil},
		},
	}

	err := Start(context.Background(), mp, mr)
	require.NoError(t, err)

	// Verify generated files.
	assert.FileExists(t, filepath.Join(hubDir, "docker-compose.yml"))
	assert.FileExists(t, filepath.Join(hubDir, "prometheus.yml"))
	assert.FileExists(t, filepath.Join(hubDir, "grafana", "provisioning", "datasources", "datasource.yml"))
	assert.FileExists(t, filepath.Join(hubDir, "grafana", "provisioning", "dashboards", "dashboards.yml"))
	assert.FileExists(t, filepath.Join(hubDir, "grafana", "dashboards", "unified-overview.json"))

	// Verify prometheus.yml contains both agents.
	promData, err := os.ReadFile(filepath.Join(hubDir, "prometheus.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(promData), "agent-a")
	assert.Contains(t, string(promData), "agent-b")
	assert.Contains(t, string(promData), "9090")
	assert.Contains(t, string(promData), "9091")
}

func TestStart_HubAlreadyRunning(t *testing.T) {
	_, hubDir := setupTestHub(t)

	dirA := t.TempDir()
	require.NoError(t, registry.Register("agent-a", dirA, 9090, 8000))

	// Create compose file to simulate existing hub.
	require.NoError(t, os.MkdirAll(hubDir, 0o755))
	composePath := filepath.Join(hubDir, "docker-compose.yml")
	require.NoError(t, os.WriteFile(composePath, []byte("version: '3'"), 0o644))

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"compose -f " + composePath + " -p volra-hub ps --quiet": {Output: "abc123", Err: nil},
		},
	}

	err := Start(context.Background(), mp, mr)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeHubAlreadyRunning, ue.Code)
}

func TestStop_NotRunning(t *testing.T) {
	setupTestHub(t)

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{}

	err := Stop(context.Background(), mp, mr)
	require.NoError(t, err)
	assert.Contains(t, mp.ResultCalls[0], "not running")
}

func TestStop_RunningHub(t *testing.T) {
	_, hubDir := setupTestHub(t)

	require.NoError(t, os.MkdirAll(hubDir, 0o755))
	composePath := filepath.Join(hubDir, "docker-compose.yml")
	require.NoError(t, os.WriteFile(composePath, []byte("version: '3'"), 0o644))

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"compose -f " + composePath + " -p volra-hub down": {Output: "", Err: nil},
		},
	}

	err := Stop(context.Background(), mp, mr)
	require.NoError(t, err)
	assert.Contains(t, mp.ResultCalls[0], "stopped")
}

func TestStatus_NoAgents(t *testing.T) {
	setupTestHub(t)

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{}

	err := Status(context.Background(), mp, mr)
	require.NoError(t, err)
	assert.Contains(t, mp.ResultCalls[0], "No agents registered")
}

func TestStatus_WithAgents(t *testing.T) {
	setupTestHub(t)

	dirA := t.TempDir()
	dirB := t.TempDir()
	require.NoError(t, registry.Register("agent-a", dirA, 9090, 8000))
	require.NoError(t, registry.Register("agent-b", dirB, 9091, 8001))

	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{}

	err := Status(context.Background(), mp, mr)
	require.NoError(t, err)
	assert.Len(t, mp.ResultCalls, 2)
	assert.Contains(t, mp.ResultCalls[0], "agent-a")
	assert.Contains(t, mp.ResultCalls[1], "agent-b")
}
