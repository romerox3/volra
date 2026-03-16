package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestParseThreeTierNamespace(t *testing.T) {
	tests := []struct {
		input  string
		server string
		agent  string
		tool   string
		wantOk bool
	}{
		{"staging/analyst/summarize", "staging", "analyst", "summarize", true},
		{"prod/my-agent/my-tool", "prod", "my-agent", "my-tool", true},
		{"agent/tool", "", "", "", false},
		{"no-separator", "", "", "", false},
		{"//empty", "", "", "", false},
		{"/agent/tool", "", "", "", false},
		{"server/agent/", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			server, agent, tool, ok := ParseThreeTierNamespace(tt.input)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.server, server)
				assert.Equal(t, tt.agent, agent)
				assert.Equal(t, tt.tool, tool)
			}
		})
	}
}

func TestIsRemoteNamespace(t *testing.T) {
	assert.True(t, IsRemoteNamespace("staging/agent/tool"))
	assert.False(t, IsRemoteNamespace("agent/tool"))
	assert.False(t, IsRemoteNamespace("tool"))
}

func TestRouter_Dispatch_RemoteTool(t *testing.T) {
	// Start a fake A2A server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/a2a", r.URL.Path)
		resp := a2aResponse{
			JSONRPC: "2.0",
			ID:      "1",
			Result: &a2aTaskResult{
				ID:     "task-1",
				Status: a2aTaskStatus{State: "completed"},
				Artifacts: []a2aArtifact{
					{Parts: []a2aPart{{Type: "text", Text: "remote result"}}},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "agent-a", OriginalName: "volra_deploy", Tool: mcp.Tool{Name: "agent-a/volra_deploy"}},
			{
				AgentName:    "analyst",
				OriginalName: "summarize",
				Tool:         mcp.Tool{Name: "staging/analyst/summarize"},
				Server:       "staging",
				AgentURL:     srv.URL,
				Remote:       true,
			},
		},
	}
	backend := &mockBackend{}
	dirs := map[string]string{"agent-a": "/tmp/agent-a"}
	router := NewRouter(cat, backend, dirs)

	// Remote tool should use A2A backend.
	result, err := router.Dispatch(context.Background(), "staging/analyst/summarize", json.RawMessage(`{"text":"hello"}`))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "remote result", result.Content[0].Text)
	// Subprocess backend should NOT have been called.
	assert.Empty(t, backend.lastDir)

	// Local tool should still use subprocess backend.
	result, err = router.Dispatch(context.Background(), "agent-a/volra_deploy", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "/tmp/agent-a", backend.lastDir)
}

func TestRouter_Dispatch_RemoteToolUnreachable(t *testing.T) {
	cat := &Catalog{
		tools: []NamespacedTool{
			{
				AgentName:    "analyst",
				OriginalName: "summarize",
				Tool:         mcp.Tool{Name: "staging/analyst/summarize"},
				Server:       "staging",
				AgentURL:     "http://192.0.2.1:9999", // unreachable
				Remote:       true,
			},
		},
	}
	router := NewRouter(cat, &mockBackend{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately to trigger fast failure

	_, err := router.Dispatch(ctx, "staging/analyst/summarize", nil)
	require.Error(t, err)
}
