package controlplane

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewStore_CreatesDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sub", "test.db")
	s, err := NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()

	assert.FileExists(t, dbPath)
}

func TestUpsertAgent_InsertAndUpdate(t *testing.T) {
	s := newTestStore(t)

	a := Agent{
		Name:      "test-agent",
		Dir:       "/tmp/test",
		Framework: "generic",
		Port:      8000,
		Status:    "unknown",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.UpsertAgent(a))

	// Verify insert.
	got, err := s.GetAgent("test-agent")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "test-agent", got.Name)
	assert.Equal(t, "generic", got.Framework)
	assert.Equal(t, 8000, got.Port)

	// Update status.
	a.Status = "healthy"
	require.NoError(t, s.UpsertAgent(a))

	got, err = s.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, "healthy", got.Status)
}

func TestGetAgent_NotFound(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetAgent("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListAgents_OrderedByName(t *testing.T) {
	s := newTestStore(t)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		require.NoError(t, s.UpsertAgent(Agent{Name: name, Dir: "/tmp", CreatedAt: time.Now().UTC()}))
	}

	agents, err := s.ListAgents()
	require.NoError(t, err)
	require.Len(t, agents, 3)
	assert.Equal(t, "alpha", agents[0].Name)
	assert.Equal(t, "bravo", agents[1].Name)
	assert.Equal(t, "charlie", agents[2].Name)
}

func TestDeleteAgent(t *testing.T) {
	s := newTestStore(t)

	require.NoError(t, s.UpsertAgent(Agent{Name: "to-delete", Dir: "/tmp", CreatedAt: time.Now().UTC()}))
	require.NoError(t, s.DeleteAgent("to-delete"))

	got, err := s.GetAgent("to-delete")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUpdateAgentStatus(t *testing.T) {
	s := newTestStore(t)

	require.NoError(t, s.UpsertAgent(Agent{Name: "a", Dir: "/tmp", Status: "unknown", CreatedAt: time.Now().UTC()}))
	require.NoError(t, s.UpdateAgentStatus("a", "healthy"))

	got, err := s.GetAgent("a")
	require.NoError(t, err)
	assert.Equal(t, "healthy", got.Status)
}

func TestImportFromLegacyRegistry(t *testing.T) {
	s := newTestStore(t)

	// Create legacy registry file.
	entries := []legacyRegistryEntry{
		{Name: "agent-a", Dir: "/opt/a", PrometheusPort: 9090, AgentPort: 8000},
		{Name: "agent-b", Dir: "/opt/b", PrometheusPort: 9091, AgentPort: 8001},
	}
	data, _ := json.Marshal(entries)
	registryPath := filepath.Join(t.TempDir(), "agents.json")
	require.NoError(t, os.WriteFile(registryPath, data, 0o644))

	n, err := s.ImportFromLegacyRegistry(registryPath)
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	// Verify imported.
	agents, err := s.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 2)

	// Import again — should not duplicate.
	n2, err := s.ImportFromLegacyRegistry(registryPath)
	require.NoError(t, err)
	assert.Equal(t, 0, n2)
}

func TestImportFromLegacyRegistry_WrappedFormat(t *testing.T) {
	s := newTestStore(t)

	// Create legacy registry in {"agents": [...]} format with project_dir field.
	wrapped := legacyRegistryFile{
		Agents: []legacyRegistryEntry{
			{Name: "cortex-agent", Dir: "/opt/cortex", PrometheusPort: 9090, AgentPort: 8000},
		},
	}
	data, _ := json.Marshal(wrapped)
	registryPath := filepath.Join(t.TempDir(), "agents.json")
	require.NoError(t, os.WriteFile(registryPath, data, 0o644))

	n, err := s.ImportFromLegacyRegistry(registryPath)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	agents, err := s.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "cortex-agent", agents[0].Name)
	assert.Equal(t, "/opt/cortex", agents[0].Dir)
}

func TestImportFromLegacyRegistry_NoFile(t *testing.T) {
	s := newTestStore(t)

	n, err := s.ImportFromLegacyRegistry("/nonexistent/agents.json")
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestAPIKey_InsertListRevoke(t *testing.T) {
	s := newTestStore(t)

	key := APIKey{
		ID:        "key-1",
		Name:      "ci-pipeline",
		KeyHash:   "$2a$10$fakehash",
		Role:      "operator",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.InsertAPIKey(key))

	// List.
	keys, err := s.ListAPIKeys()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "ci-pipeline", keys[0].Name)
	assert.Equal(t, "operator", keys[0].Role)

	// HasAPIKeys.
	has, err := s.HasAPIKeys()
	require.NoError(t, err)
	assert.True(t, has)

	// Revoke.
	require.NoError(t, s.RevokeAPIKey("key-1"))

	// Active keys should be empty.
	active, err := s.GetActiveAPIKeys()
	require.NoError(t, err)
	assert.Empty(t, active)

	// HasAPIKeys should be false.
	has, err = s.HasAPIKeys()
	require.NoError(t, err)
	assert.False(t, has)
}

func TestAPIKey_RevokeNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.RevokeAPIKey("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or already revoked")
}

func TestAPIKey_InvalidRole(t *testing.T) {
	s := newTestStore(t)

	key := APIKey{
		ID:        "key-bad",
		Name:      "bad",
		KeyHash:   "hash",
		Role:      "superadmin",
		CreatedAt: time.Now().UTC(),
	}
	err := s.InsertAPIKey(key)
	assert.Error(t, err) // CHECK constraint violation
}

func TestFederationPeer_InsertListDelete(t *testing.T) {
	s := newTestStore(t)

	peer := FederationPeer{
		URL:     "https://staging.example.com:4441",
		Name:    "staging",
		APIKey:  "secret-key",
		AddedAt: time.Now().UTC(),
	}
	require.NoError(t, s.InsertPeer(peer))

	// List.
	peers, err := s.ListPeers()
	require.NoError(t, err)
	require.Len(t, peers, 1)
	assert.Equal(t, "staging", peers[0].Name)

	// Delete.
	require.NoError(t, s.DeletePeer("https://staging.example.com:4441"))

	peers, err = s.ListPeers()
	require.NoError(t, err)
	assert.Empty(t, peers)
}

func TestFederationPeer_DeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeletePeer("https://nonexistent.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetAPIKeyByID(t *testing.T) {
	s := newTestStore(t)

	key := APIKey{
		ID:        "key-lookup",
		Name:      "test",
		KeyHash:   "hash123",
		Role:      "viewer",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.InsertAPIKey(key))

	got, err := s.GetAPIKeyByID("key-lookup")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "viewer", got.Role)

	// Not found.
	got, err = s.GetAPIKeyByID("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListAgents_Empty(t *testing.T) {
	s := newTestStore(t)

	agents, err := s.ListAgents()
	require.NoError(t, err)
	assert.Nil(t, agents)
}
