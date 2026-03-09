package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestServer() (*Server, *mockBackend) {
	cat := &Catalog{
		tools: []NamespacedTool{
			{AgentName: "agent-a", OriginalName: "volra_deploy", Tool: mcp.Tool{
				Name:        "agent-a/volra_deploy",
				Description: "[agent-a] Deploy agent",
				InputSchema: map[string]any{"type": "object"},
			}},
		},
	}
	backend := &mockBackend{}
	dirs := map[string]string{"agent-a": "/tmp/agent-a"}
	router := NewRouter(cat, backend, dirs)
	return NewServer(router, "0.6.0-test"), backend
}

func postMCP(srv *Server, req mcp.Request) *httptest.ResponseRecorder {
	return postMCPWithSession(srv, req, "")
}

func postMCPWithSession(srv *Server, req mcp.Request, sessionID string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		r.Header.Set(SessionHeader, sessionID)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func TestServer_Initialize(t *testing.T) {
	srv, _ := buildTestServer()

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test"}}`),
	}
	w := postMCP(srv, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	// Parse result.
	resultBytes, _ := json.Marshal(resp.Result)
	var initResult mcp.InitializeResult
	require.NoError(t, json.Unmarshal(resultBytes, &initResult))
	assert.Equal(t, "volra-gateway", initResult.ServerInfo.Name)
	assert.Equal(t, "0.6.0-test", initResult.ServerInfo.Version)
}

func TestServer_ToolsList(t *testing.T) {
	srv, _ := buildTestServer()

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}
	w := postMCP(srv, req)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult mcp.ToolsListResult
	require.NoError(t, json.Unmarshal(resultBytes, &toolsResult))
	assert.Len(t, toolsResult.Tools, 1)
	assert.Equal(t, "agent-a/volra_deploy", toolsResult.Tools[0].Name)
}

func TestServer_ToolsCall(t *testing.T) {
	srv, backend := buildTestServer()
	backend.result = mcp.SuccessResult("deployed successfully")

	params, _ := json.Marshal(mcp.ToolCallParams{
		Name:      "agent-a/volra_deploy",
		Arguments: json.RawMessage(`{"force": true}`),
	})
	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params:  json.RawMessage(params),
	}
	w := postMCP(srv, req)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	resultBytes, _ := json.Marshal(resp.Result)
	var toolResult mcp.ToolCallResult
	require.NoError(t, json.Unmarshal(resultBytes, &toolResult))
	assert.False(t, toolResult.IsError)
	assert.Equal(t, "deployed successfully", toolResult.Content[0].Text)

	// Verify routing.
	assert.Equal(t, "/tmp/agent-a", backend.lastDir)
	assert.Equal(t, "volra_deploy", backend.lastParams.Name)
}

func TestServer_UnknownMethod(t *testing.T) {
	srv, _ := buildTestServer()

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "resources/list",
	}
	w := postMCP(srv, req)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, mcp.CodeMethodNotFound, resp.Error.Code)
}

func TestServer_Health(t *testing.T) {
	srv, _ := buildTestServer()

	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, float64(1), body["tools"])
}

func TestServer_InvalidJSON(t *testing.T) {
	srv, _ := buildTestServer()

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{invalid`)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, mcp.CodeParseError, resp.Error.Code)
}

func TestServer_SessionHeaderReturned(t *testing.T) {
	srv, _ := buildTestServer()

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}

	// First request — new session created.
	w1 := postMCP(srv, req)
	sessionID := w1.Header().Get(SessionHeader)
	assert.NotEmpty(t, sessionID, "server should return Mcp-Session-Id")

	// Second request with same session — reuses session.
	w2 := postMCPWithSession(srv, req, sessionID)
	assert.Equal(t, sessionID, w2.Header().Get(SessionHeader))
}

func TestServer_SessionRecordsAgentInteraction(t *testing.T) {
	srv, backend := buildTestServer()
	backend.result = mcp.SuccessResult("ok")

	// Initialize to get session ID.
	initReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}
	w := postMCP(srv, initReq)
	sessionID := w.Header().Get(SessionHeader)

	// Call a tool with the session.
	params, _ := json.Marshal(mcp.ToolCallParams{
		Name: "agent-a/volra_deploy",
	})
	callReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/call",
		Params:  json.RawMessage(params),
	}
	postMCPWithSession(srv, callReq, sessionID)

	// Verify session recorded agent-a interaction.
	session, isNew := srv.sessions.GetOrCreate(sessionID)
	assert.False(t, isNew)
	assert.True(t, session.AgentBackends["agent-a"])
}
