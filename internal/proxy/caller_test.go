package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCaller_Default(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ask", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "hello", req["question"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "world"})
	}))
	defer agent.Close()

	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDefault, nil)
	result, err := caller.Call("", "hello", nil)
	require.NoError(t, err)
	assert.Equal(t, "world", result)
}

func TestAgentCaller_Default_EmptyMode(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "ok"})
	}))
	defer agent.Close()

	caller := NewAgentCaller(agent.URL, "", nil)
	result, err := caller.Call("", "test", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestAgentCaller_Declarative(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "test query", req["query"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"documents": "doc1, doc2"})
	}))
	defer agent.Close()

	skills := []agentfile.A2ASkill{
		{
			ID:            "search",
			Name:          "semantic-search",
			Endpoint:      "/search",
			RequestField:  "query",
			ResponseField: "documents",
		},
	}
	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDeclarative, skills)
	result, err := caller.Call("search", "test query", nil)
	require.NoError(t, err)
	assert.Equal(t, "doc1, doc2", result)
}

func TestAgentCaller_Declarative_DefaultFields(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ask", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "hello", req["question"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": "hi"})
	}))
	defer agent.Close()

	skills := []agentfile.A2ASkill{
		{ID: "chat", Name: "chat", Endpoint: "/ask"},
	}
	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDeclarative, skills)
	result, err := caller.Call("chat", "hello", nil)
	require.NoError(t, err)
	assert.Equal(t, "hi", result)
}

func TestAgentCaller_Declarative_SkillNotFound(t *testing.T) {
	caller := NewAgentCaller("http://localhost:9999", agentfile.A2AModeDeclarative, nil)
	_, err := caller.Call("nonexistent", "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill")
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentCaller_Passthrough(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/a2a", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body) // echo back
	}))
	defer agent.Close()

	caller := NewAgentCaller(agent.URL, agentfile.A2AModePassthrough, nil)
	raw := []byte(`{"jsonrpc":"2.0","method":"Tasks/send","params":{}}`)
	result, err := caller.Call("", "", raw)
	require.NoError(t, err)
	assert.Contains(t, result, "Tasks/send")
}

func TestAgentCaller_AgentUnreachable(t *testing.T) {
	caller := NewAgentCaller("http://localhost:1", agentfile.A2AModeDefault, nil)
	_, err := caller.Call("", "hello", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent unreachable")
}

func TestAgentCaller_AgentErrorStatus(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer agent.Close()

	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDefault, nil)
	_, err := caller.Call("", "hello", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAgentCaller_NonJSONResponse(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer agent.Close()

	caller := NewAgentCaller(agent.URL, agentfile.A2AModeDefault, nil)
	result, err := caller.Call("", "hello", nil)
	require.NoError(t, err)
	assert.Equal(t, "plain text response", result)
}
