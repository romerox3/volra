package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBackend implements Backend for testing.
type mockBackend struct {
	lastDir    string
	lastParams mcp.ToolCallParams
	result     *mcp.ToolCallResult
	err        error
}

func (m *mockBackend) Call(_ context.Context, agentDir string, params mcp.ToolCallParams) (*mcp.ToolCallResult, error) {
	m.lastDir = agentDir
	m.lastParams = params
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return mcp.SuccessResult("ok"), nil
}

func buildTestRouter() (*Router, *mockBackend) {
	cat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "agent-a", OriginalName: "volra_deploy", Tool: mcp.Tool{Name: "agent-a/volra_deploy"}},
			{AgentName: "agent-a", OriginalName: "volra_status", Tool: mcp.Tool{Name: "agent-a/volra_status"}},
			{AgentName: "agent-b", OriginalName: "volra_deploy", Tool: mcp.Tool{Name: "agent-b/volra_deploy"}},
		},
	}
	backend := &mockBackend{}
	dirs := map[string]string{
		"agent-a": "/tmp/agent-a",
		"agent-b": "/tmp/agent-b",
	}
	return NewRouter(cat, backend, dirs), backend
}

func TestRouter_Dispatch_RoutesToCorrectAgent(t *testing.T) {
	router, backend := buildTestRouter()
	args := json.RawMessage(`{"force": true}`)

	result, err := router.Dispatch(context.Background(), "agent-a/volra_deploy", args)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	assert.Equal(t, "/tmp/agent-a", backend.lastDir)
	assert.Equal(t, "volra_deploy", backend.lastParams.Name, "should strip namespace")
	assert.Equal(t, args, backend.lastParams.Arguments)
}

func TestRouter_Dispatch_AgentB(t *testing.T) {
	router, backend := buildTestRouter()

	result, err := router.Dispatch(context.Background(), "agent-b/volra_deploy", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "/tmp/agent-b", backend.lastDir)
}

func TestRouter_Dispatch_UnknownTool(t *testing.T) {
	router, _ := buildTestRouter()

	result, err := router.Dispatch(context.Background(), "agent-c/volra_deploy", nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Unknown tool")
}

func TestRouter_Dispatch_BackendError(t *testing.T) {
	router, backend := buildTestRouter()
	backend.err = fmt.Errorf("subprocess crashed")

	_, err := router.Dispatch(context.Background(), "agent-a/volra_deploy", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subprocess crashed")
}

func TestRouter_ListTools(t *testing.T) {
	router, _ := buildTestRouter()

	tools := router.ListTools()
	assert.Len(t, tools, 3)
	assert.Equal(t, "agent-a/volra_deploy", tools[0].Name)
	assert.Equal(t, "agent-b/volra_deploy", tools[2].Name)
}

func TestRouter_ReloadCatalog(t *testing.T) {
	router, _ := buildTestRouter()

	newCat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "agent-c", OriginalName: "volra_logs", Tool: mcp.Tool{Name: "agent-c/volra_logs"}},
		},
	}
	newDirs := map[string]string{"agent-c": "/tmp/agent-c"}
	router.ReloadCatalog(newCat, newDirs)

	tools := router.ListTools()
	assert.Len(t, tools, 1)
	assert.Equal(t, "agent-c/volra_logs", tools[0].Name)
}

func TestParseNamespace(t *testing.T) {
	tests := []struct {
		input     string
		agent     string
		tool      string
		wantOk    bool
	}{
		{"agent-a/volra_deploy", "agent-a", "volra_deploy", true},
		{"my-agent/my-tool", "my-agent", "my-tool", true},
		{"no-separator", "", "", false},
		{"/leading-slash", "", "", false},
		{"trailing/", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			agent, tool, ok := ParseNamespace(tt.input)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.agent, agent)
				assert.Equal(t, tt.tool, tool)
			}
		})
	}
}
