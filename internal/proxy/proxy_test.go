package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/a2a"
	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProxy(t *testing.T, agentHandler http.HandlerFunc) (*Handler, func()) {
	t.Helper()
	agent := httptest.NewServer(agentHandler)

	cardDir := t.TempDir()
	cardPath := filepath.Join(cardDir, "agent-card.json")
	card := `{"name":"test-agent","url":"http://localhost:8000"}`
	require.NoError(t, os.WriteFile(cardPath, []byte(card), 0644))

	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDefault, nil)
	handler, err := NewHandler(agent.URL, cardPath, caller)
	require.NoError(t, err)

	return handler, agent.Close
}

func sendRPC(t *testing.T, handler http.Handler, method string, params interface{}) *httptest.ResponseRecorder {
	t.Helper()
	paramsJSON, _ := json.Marshal(params)
	body := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  method,
		Params:  paramsJSON,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestProxy_ServesCard(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "test-agent")
}

func TestProxy_CardNotFound(t *testing.T) {
	caller := NewAgentCaller("http://localhost:1", agentfile.A2AModeDefault, nil)
	handler, err := NewHandler("http://localhost:1", "/nonexistent/card.json", caller)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxy_TasksSend(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "Hello from agent"})
	})
	defer cleanup()

	params := a2a.TaskSendParams{
		Message: a2a.Message{
			Role:  "user",
			Parts: []a2a.Part{{Type: "text", Text: "do something"}},
		},
	}
	w := sendRPC(t, handler, "Tasks/send", params)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	resultJSON, _ := json.Marshal(resp.Result)
	var task a2a.Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))
	assert.Equal(t, a2a.TaskStateCompleted, task.Status.State)
	require.Len(t, task.Artifacts, 1)
	assert.Equal(t, "Hello from agent", task.Artifacts[0].Parts[0].Text)
}

func TestProxy_TasksGet(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "ok"})
	})
	defer cleanup()

	// Create a task first.
	params := a2a.TaskSendParams{
		Message: a2a.Message{
			Role:  "user",
			Parts: []a2a.Part{{Type: "text", Text: "test"}},
		},
	}
	w := sendRPC(t, handler, "Tasks/send", params)

	var sendResp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sendResp))
	resultJSON, _ := json.Marshal(sendResp.Result)
	var task a2a.Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))

	// Now get it.
	w2 := sendRPC(t, handler, "Tasks/get", a2a.TaskGetParams{ID: task.ID})

	var getResp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &getResp))
	assert.Nil(t, getResp.Error)
}

func TestProxy_TasksGetNotFound(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {})
	defer cleanup()

	w := sendRPC(t, handler, "Tasks/get", a2a.TaskGetParams{ID: "nonexistent"})

	var resp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32001, resp.Error.Code)
}

func TestProxy_UnknownMethod(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {})
	defer cleanup()

	w := sendRPC(t, handler, "Unknown/method", nil)

	var resp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestProxy_InvalidJSON(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32700, resp.Error.Code)
}

func TestProxy_ReverseProxyFallback(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
		}
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "healthy", w.Body.String())
}

func TestProxy_TasksCancel(t *testing.T) {
	handler, cleanup := setupProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "ok"})
	})
	defer cleanup()

	// Create a task.
	params := a2a.TaskSendParams{
		Message: a2a.Message{Role: "user", Parts: []a2a.Part{{Type: "text", Text: "test"}}},
	}
	w := sendRPC(t, handler, "Tasks/send", params)
	var sendResp a2a.JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sendResp))
	resultJSON, _ := json.Marshal(sendResp.Result)
	var task a2a.Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))

	// Cancel it.
	w2 := sendRPC(t, handler, "Tasks/cancel", a2a.TaskGetParams{ID: task.ID})
	assert.Equal(t, http.StatusOK, w2.Code)
}
