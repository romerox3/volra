package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestA2ABackend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/a2a", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req a2aRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "Tasks/send", req.Method)

		resp := a2aResponse{
			JSONRPC: "2.0",
			ID:      "1",
			Result: &a2aTaskResult{
				ID:     "task-1",
				Status: a2aTaskStatus{State: "completed"},
				Artifacts: []a2aArtifact{
					{Parts: []a2aPart{{Type: "text", Text: "Hello from remote agent"}}},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	backend := NewA2ABackend()
	result, err := backend.CallRemote(context.Background(), srv.URL, "summarize", json.RawMessage(`{"text":"hello"}`))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "Hello from remote agent", result.Content[0].Text)
}

func TestA2ABackend_RemoteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := a2aResponse{
			JSONRPC: "2.0",
			ID:      "1",
			Error:   &a2aResponseError{Code: -32000, Message: "tool not found"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	backend := NewA2ABackend()
	_, err := backend.CallRemote(context.Background(), srv.URL, "unknown", nil)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeA2ARemoteCallFailed, ue.Code)
	assert.Contains(t, ue.What, "tool not found")
}

func TestA2ABackend_FailedTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := a2aResponse{
			JSONRPC: "2.0",
			ID:      "1",
			Result:  &a2aTaskResult{ID: "t1", Status: a2aTaskStatus{State: "failed"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	backend := NewA2ABackend()
	result, err := backend.CallRemote(context.Background(), srv.URL, "tool", nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "failed")
}

func TestA2ABackend_Unreachable(t *testing.T) {
	backend := NewA2ABackend()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := backend.CallRemote(ctx, "http://192.0.2.1:9999", "tool", nil)
	require.Error(t, err)
}

func TestA2ABackend_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	backend := NewA2ABackend()
	_, err := backend.CallRemote(context.Background(), srv.URL, "tool", nil)
	require.Error(t, err)
}

func TestA2ABackend_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := a2aResponse{
			JSONRPC: "2.0",
			ID:      "1",
			Result:  &a2aTaskResult{ID: "t1", Status: a2aTaskStatus{State: "completed"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	backend := NewA2ABackend()
	result, err := backend.CallRemote(context.Background(), srv.URL, "tool", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "task completed", result.Content[0].Text)
}
