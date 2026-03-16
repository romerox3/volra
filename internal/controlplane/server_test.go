package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*Server, *Store) {
	t.Helper()
	s := newTestStore(t)
	srv := NewServer(s, 0)
	return srv, s
}

func TestHandleHealth(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestHandleListAgents_Empty(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var agents []Agent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &agents))
	assert.Empty(t, agents)
}

func TestHandleListAgents_WithAgents(t *testing.T) {
	srv, store := newTestServer(t)

	require.NoError(t, store.UpsertAgent(Agent{Name: "alpha", Dir: "/tmp/a", Status: "healthy", CreatedAt: time.Now().UTC()}))
	require.NoError(t, store.UpsertAgent(Agent{Name: "bravo", Dir: "/tmp/b", Status: "unknown", CreatedAt: time.Now().UTC()}))

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var agents []Agent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &agents))
	assert.Len(t, agents, 2)
	assert.Equal(t, "alpha", agents[0].Name)
}

func TestHandleGetAgent_Found(t *testing.T) {
	srv, store := newTestServer(t)

	require.NoError(t, store.UpsertAgent(Agent{
		Name:       "test-agent",
		Dir:        "/opt/agents/test",
		Framework:  "langchain",
		Port:       8000,
		HealthPath: "/health",
		Status:     "healthy",
		CreatedAt:  time.Now().UTC(),
	}))

	req := httptest.NewRequest("GET", "/api/agents/test-agent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var agent Agent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &agent))
	assert.Equal(t, "test-agent", agent.Name)
	assert.Equal(t, "langchain", agent.Framework)
	assert.Equal(t, 8000, agent.Port)
}

func TestHandleGetAgent_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleDeployAgent(t *testing.T) {
	srv, store := newTestServer(t)

	require.NoError(t, store.UpsertAgent(Agent{Name: "deploy-me", Dir: "/tmp", Status: "healthy", CreatedAt: time.Now().UTC()}))

	req := httptest.NewRequest("POST", "/api/agents/deploy-me/deploy", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	// Verify status changed.
	agent, err := store.GetAgent("deploy-me")
	require.NoError(t, err)
	assert.Equal(t, "deploying", agent.Status)
}

func TestHandleDeployAgent_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("POST", "/api/agents/nonexistent/deploy", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleStopAgent(t *testing.T) {
	srv, store := newTestServer(t)

	require.NoError(t, store.UpsertAgent(Agent{Name: "stop-me", Dir: "/tmp", Status: "healthy", CreatedAt: time.Now().UTC()}))

	req := httptest.NewRequest("POST", "/api/agents/stop-me/stop", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	agent, err := store.GetAgent("stop-me")
	require.NoError(t, err)
	assert.Equal(t, "stopped", agent.Status)
}

func TestHandleStopAgent_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("POST", "/api/agents/nonexistent/stop", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleFederationCapabilities_NoPeers(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/federation/capabilities", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var caps []FederatedCapability
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &caps))
	assert.Empty(t, caps)
}

func TestHandleFederationCapabilities_WithLocalAgents(t *testing.T) {
	srv, store := newTestServer(t)

	// Create a mock A2A card server for the local agent.
	cardSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-card.json" {
			json.NewEncoder(w).Encode(map[string]string{
				"name":            "local-agent",
				"documentVersion": "0.3.0",
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer cardSrv.Close()

	// We need to get the port from the card server to set as agent port.
	// Use a port guaranteed to have nothing listening (ephemeral range, unlikely conflict).
	require.NoError(t, store.UpsertAgent(Agent{
		Name:      "local-agent",
		Dir:       "/tmp/a",
		Status:    "healthy",
		Port:      59123,
		CreatedAt: time.Now().UTC(),
	}))

	// The federation endpoint fetches agent cards via HTTP.
	// Since nothing listens on port 59123, the card fetch fails → card_error.
	req := httptest.NewRequest("GET", "/api/federation/capabilities", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var caps []FederatedCapability
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &caps))
	require.Len(t, caps, 1)
	assert.Equal(t, "local-agent", caps[0].Agent)
	assert.Equal(t, "local", caps[0].Server)
	assert.Equal(t, "card_error", caps[0].Status)
}

func TestContentTypeJSON(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
