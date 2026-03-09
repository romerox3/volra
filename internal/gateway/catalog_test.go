package gateway

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSpawner implements MCPSpawner for testing.
type mockSpawner struct {
	// toolsByDir maps agent directory to its tool list.
	toolsByDir map[string][]mcp.Tool
	// errByDir maps agent directory to an error.
	errByDir map[string]error
}

func (m *mockSpawner) DiscoverTools(_ context.Context, agentDir string) ([]mcp.Tool, error) {
	if err, ok := m.errByDir[agentDir]; ok {
		return nil, err
	}
	tools, ok := m.toolsByDir[agentDir]
	if !ok {
		return nil, fmt.Errorf("no tools configured for %s", agentDir)
	}
	return tools, nil
}

func TestBuildCatalog_TwoAgents(t *testing.T) {
	agents := []registry.AgentEntry{
		{Name: "agent-a", ProjectDir: "/tmp/agent-a"},
		{Name: "agent-b", ProjectDir: "/tmp/agent-b"},
	}

	spawner := &mockSpawner{
		toolsByDir: map[string][]mcp.Tool{
			"/tmp/agent-a": {
				{Name: "volra_deploy", Description: "Deploy agent", InputSchema: map[string]any{"type": "object"}},
				{Name: "volra_status", Description: "Check status", InputSchema: map[string]any{"type": "object"}},
			},
			"/tmp/agent-b": {
				{Name: "volra_deploy", Description: "Deploy agent", InputSchema: map[string]any{"type": "object"}},
			},
		},
	}

	mp := &testutil.MockPresenter{}
	cat, err := BuildCatalog(context.Background(), agents, spawner, mp, 5*time.Second)
	require.NoError(t, err)

	tools := cat.Tools()
	assert.Len(t, tools, 3)

	// Verify namespacing.
	assert.Equal(t, "agent-a/volra_deploy", tools[0].Tool.Name)
	assert.Equal(t, "agent-a", tools[0].AgentName)
	assert.Equal(t, "volra_deploy", tools[0].OriginalName)

	assert.Equal(t, "agent-a/volra_status", tools[1].Tool.Name)
	assert.Equal(t, "agent-b/volra_deploy", tools[2].Tool.Name)

	// Description prefixed.
	assert.Equal(t, "[agent-a] Deploy agent", tools[0].Tool.Description)
}

func TestBuildCatalog_SkipsFailingAgent(t *testing.T) {
	agents := []registry.AgentEntry{
		{Name: "good-agent", ProjectDir: "/tmp/good"},
		{Name: "bad-agent", ProjectDir: "/tmp/bad"},
	}

	spawner := &mockSpawner{
		toolsByDir: map[string][]mcp.Tool{
			"/tmp/good": {
				{Name: "volra_deploy", Description: "Deploy", InputSchema: map[string]any{"type": "object"}},
			},
		},
		errByDir: map[string]error{
			"/tmp/bad": fmt.Errorf("connection timeout"),
		},
	}

	mp := &testutil.MockPresenter{}
	cat, err := BuildCatalog(context.Background(), agents, spawner, mp, 5*time.Second)
	require.NoError(t, err)

	tools := cat.Tools()
	assert.Len(t, tools, 1)
	assert.Equal(t, "good-agent/volra_deploy", tools[0].Tool.Name)

	// Warning emitted for failed agent.
	require.Len(t, mp.WarnCalls, 1)
	assert.Contains(t, mp.WarnCalls[0].What, "bad-agent")
	assert.Contains(t, mp.WarnCalls[0].What, "connection timeout")
}

func TestBuildCatalog_NoAgents(t *testing.T) {
	mp := &testutil.MockPresenter{}
	_, err := BuildCatalog(context.Background(), nil, nil, mp, 5*time.Second)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeGatewayNoAgents, ue.Code)
}

func TestBuildCatalog_AllAgentsFail(t *testing.T) {
	agents := []registry.AgentEntry{
		{Name: "bad-1", ProjectDir: "/tmp/bad1"},
		{Name: "bad-2", ProjectDir: "/tmp/bad2"},
	}

	spawner := &mockSpawner{
		errByDir: map[string]error{
			"/tmp/bad1": fmt.Errorf("timeout"),
			"/tmp/bad2": fmt.Errorf("crash"),
		},
	}

	mp := &testutil.MockPresenter{}
	_, err := BuildCatalog(context.Background(), agents, spawner, mp, 5*time.Second)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeGatewayToolsFailed, ue.Code)
}

func TestCatalog_Lookup(t *testing.T) {
	cat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "agent-a", OriginalName: "volra_deploy", Tool: mcp.Tool{Name: "agent-a/volra_deploy"}},
			{AgentName: "agent-b", OriginalName: "volra_status", Tool: mcp.Tool{Name: "agent-b/volra_status"}},
		},
	}

	t.Run("found", func(t *testing.T) {
		nt, ok := cat.Lookup("agent-a/volra_deploy")
		assert.True(t, ok)
		assert.Equal(t, "agent-a", nt.AgentName)
		assert.Equal(t, "volra_deploy", nt.OriginalName)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := cat.Lookup("agent-c/volra_deploy")
		assert.False(t, ok)
	})
}

func TestCatalog_ToolsReturnsCopy(t *testing.T) {
	cat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "a", Tool: mcp.Tool{Name: "a/tool"}},
		},
	}

	tools := cat.Tools()
	tools[0].AgentName = "mutated"

	// Original must be unaffected.
	assert.Equal(t, "a", cat.tools[0].AgentName)
}
