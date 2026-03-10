package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRegistry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.json")

	origPathFunc := PathFunc
	PathFunc = func() (string, error) { return path, nil }
	t.Cleanup(func() { PathFunc = origPathFunc })

	return path
}

func TestRegister_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "agents.json")

	origPathFunc := PathFunc
	PathFunc = func() (string, error) { return path, nil }
	t.Cleanup(func() { PathFunc = origPathFunc })

	projectDir := t.TempDir()
	err := Register("my-agent", projectDir, 9090, 8000)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err, "registry file should exist")

	agents, err := List()
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "my-agent", agents[0].Name)
	assert.Equal(t, 9090, agents[0].PrometheusPort)
	assert.Equal(t, 8000, agents[0].AgentPort)
	assert.Equal(t, "deployed", agents[0].Status)
}

func TestRegister_TwoAgents(t *testing.T) {
	setupTestRegistry(t)

	dirA := t.TempDir()
	dirB := t.TempDir()
	require.NoError(t, Register("agent-a", dirA, 9090, 8000))
	require.NoError(t, Register("agent-b", dirB, 9091, 8001))

	agents, err := List()
	require.NoError(t, err)
	assert.Len(t, agents, 2)
}

func TestRegister_UpdateExisting(t *testing.T) {
	setupTestRegistry(t)

	projectDir := t.TempDir()
	absDir, _ := filepath.Abs(projectDir)

	require.NoError(t, Register("my-agent", projectDir, 9090, 8000))
	require.NoError(t, Register("my-agent", projectDir, 9091, 8001))

	agents, err := List()
	require.NoError(t, err)
	require.Len(t, agents, 1, "should update, not duplicate")
	assert.Equal(t, 9091, agents[0].PrometheusPort)
	assert.Equal(t, 8001, agents[0].AgentPort)
	assert.Equal(t, absDir, agents[0].ProjectDir)
}

func TestDeregister_RemovesAgent(t *testing.T) {
	setupTestRegistry(t)

	dirA := t.TempDir()
	dirB := t.TempDir()
	require.NoError(t, Register("agent-a", dirA, 9090, 8000))
	require.NoError(t, Register("agent-b", dirB, 9091, 8001))

	err := Deregister("agent-a")
	require.NoError(t, err)

	agents, err := List()
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "agent-b", agents[0].Name)
}

func TestDeregister_NonexistentIsNoOp(t *testing.T) {
	setupTestRegistry(t)

	projectDir := t.TempDir()
	require.NoError(t, Register("my-agent", projectDir, 9090, 8000))

	err := Deregister("nonexistent")
	require.NoError(t, err)

	agents, err := List()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
}

func TestList_EmptyWhenNoFile(t *testing.T) {
	setupTestRegistry(t)

	agents, err := List()
	require.NoError(t, err)
	assert.Empty(t, agents)
}

func TestList_ReadsExistingFile(t *testing.T) {
	path := setupTestRegistry(t)

	projectDir := t.TempDir()
	reg := Registry{
		Agents: []AgentEntry{
			{Name: "pre-existing", ProjectDir: projectDir, PrometheusPort: 9090, AgentPort: 8000, Status: "deployed"},
		},
	}
	data, _ := json.MarshalIndent(reg, "", "  ")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	agents, err := List()
	require.NoError(t, err)
	require.Len(t, agents, 1)
	assert.Equal(t, "pre-existing", agents[0].Name)
}

func TestList_CleansStaleEntries(t *testing.T) {
	path := setupTestRegistry(t)

	// Create a real dir and a fake dir
	realDir := t.TempDir()
	fakeDir := filepath.Join(t.TempDir(), "nonexistent-subdir")

	reg := Registry{
		Agents: []AgentEntry{
			{Name: "alive", ProjectDir: realDir, PrometheusPort: 9090, AgentPort: 8000, Status: "deployed"},
			{Name: "ghost", ProjectDir: fakeDir, PrometheusPort: 9091, AgentPort: 8001, Status: "deployed"},
		},
	}
	data, _ := json.MarshalIndent(reg, "", "  ")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	agents, err := List()
	require.NoError(t, err)
	require.Len(t, agents, 1, "stale entry should be removed")
	assert.Equal(t, "alive", agents[0].Name)

	// Verify cleanup was persisted
	data2, _ := os.ReadFile(path)
	var reg2 Registry
	require.NoError(t, json.Unmarshal(data2, &reg2))
	assert.Len(t, reg2.Agents, 1, "cleanup should be persisted to disk")
}

func TestAtomicWrite(t *testing.T) {
	path := setupTestRegistry(t)

	projectDir := t.TempDir()
	require.NoError(t, Register("my-agent", projectDir, 9090, 8000))

	// Verify no .tmp file remains
	_, err := os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err), "temp file should be cleaned up")

	// Verify valid JSON
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var reg Registry
	assert.NoError(t, json.Unmarshal(data, &reg))
}
