package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRemoteAgents_Success(t *testing.T) {
	agents := []Agent{
		{Name: "remote-a", Dir: "/opt/a", Status: "healthy", CreatedAt: time.Now().UTC()},
		{Name: "remote-b", Dir: "/opt/b", Status: "unknown", CreatedAt: time.Now().UTC()},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/agents", r.URL.Path)
		json.NewEncoder(w).Encode(agents)
	}))
	defer srv.Close()

	client := NewFederationClient()
	peer := FederationPeer{URL: srv.URL, Name: "staging"}

	result, err := client.FetchRemoteAgents(context.Background(), peer)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "remote-a", result[0].Name)
	assert.Equal(t, "staging", result[0].Server)
}

func TestFetchRemoteAgents_WithAPIKey(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode([]Agent{})
	}))
	defer srv.Close()

	client := NewFederationClient()
	peer := FederationPeer{URL: srv.URL, Name: "prod", APIKey: "secret-key"}

	_, err := client.FetchRemoteAgents(context.Background(), peer)
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-key", receivedAuth)
}

func TestFetchRemoteAgents_PeerDown(t *testing.T) {
	client := NewFederationClient()
	client.httpClient.Timeout = 1 * time.Second
	peer := FederationPeer{URL: "http://localhost:1", Name: "dead"}

	_, err := client.FetchRemoteAgents(context.Background(), peer)
	assert.Error(t, err)
}

func TestFetchRemoteAgents_UsesCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode([]Agent{{Name: "cached", CreatedAt: time.Now().UTC()}})
	}))
	defer srv.Close()

	client := NewFederationClient()
	peer := FederationPeer{URL: srv.URL, Name: "cache-test"}

	_, err := client.FetchRemoteAgents(context.Background(), peer)
	require.NoError(t, err)
	first := callCount

	_, err = client.FetchRemoteAgents(context.Background(), peer)
	require.NoError(t, err)
	assert.Equal(t, first, callCount, "second call should use cache")
}

func TestAggregateAgents_LocalOnly(t *testing.T) {
	client := NewFederationClient()
	local := []Agent{{Name: "local-a", CreatedAt: time.Now().UTC()}}

	result := client.AggregateAgents(context.Background(), local, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "local-a", result[0].Name)
	assert.Equal(t, "local", result[0].Server)
}

func TestAggregateAgents_WithPeers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Agent{
			{Name: "remote-x", CreatedAt: time.Now().UTC()},
		})
	}))
	defer srv.Close()

	client := NewFederationClient()
	local := []Agent{{Name: "local-a", CreatedAt: time.Now().UTC()}}
	peers := []FederationPeer{{URL: srv.URL, Name: "staging"}}

	result := client.AggregateAgents(context.Background(), local, peers)
	require.Len(t, result, 2)

	names := map[string]bool{}
	for _, a := range result {
		names[a.Name] = true
	}
	assert.True(t, names["local-a"])
	assert.True(t, names["remote-x"])
}

func TestAggregateAgents_SkipsUnreachablePeers(t *testing.T) {
	client := NewFederationClient()
	client.httpClient.Timeout = 1 * time.Second
	local := []Agent{{Name: "local", CreatedAt: time.Now().UTC()}}
	peers := []FederationPeer{{URL: "http://localhost:1", Name: "dead"}}

	result := client.AggregateAgents(context.Background(), local, peers)
	require.Len(t, result, 1)
	assert.Equal(t, "local", result[0].Name)
}

func TestCheckPeerHealth_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewFederationClient()
	err := client.CheckPeerHealth(context.Background(), srv.URL)
	assert.NoError(t, err)
}

func TestCheckPeerHealth_Failure(t *testing.T) {
	client := NewFederationClient()
	client.httpClient.Timeout = 1 * time.Second
	err := client.CheckPeerHealth(context.Background(), "http://localhost:1")
	assert.Error(t, err)
}
