package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// FederatedAgent is an agent with its origin server.
type FederatedAgent struct {
	Agent
	Server string `json:"server"`
}

// FederationClient queries remote control plane peers.
type FederationClient struct {
	httpClient *http.Client
	cache      map[string]*federationCache
	mu         sync.RWMutex
	cacheTTL   time.Duration
}

type federationCache struct {
	agents    []FederatedAgent
	fetchedAt time.Time
}

// NewFederationClient creates a federation client.
func NewFederationClient() *FederationClient {
	return &FederationClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]*federationCache),
		cacheTTL:   30 * time.Second,
	}
}

// FetchRemoteAgents queries a remote peer's /api/agents endpoint.
func (f *FederationClient) FetchRemoteAgents(ctx context.Context, peer FederationPeer) ([]FederatedAgent, error) {
	// Check cache.
	f.mu.RLock()
	if cached, ok := f.cache[peer.URL]; ok && time.Since(cached.fetchedAt) < f.cacheTTL {
		f.mu.RUnlock()
		return cached.agents, nil
	}
	f.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", peer.URL+"/api/agents", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if peer.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+peer.APIKey)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying peer %s: %w", peer.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer %s returned HTTP %d", peer.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("reading peer response: %w", err)
	}

	var agents []Agent
	if err := json.Unmarshal(body, &agents); err != nil {
		return nil, fmt.Errorf("parsing peer response: %w", err)
	}

	// Tag with server origin.
	serverName := peer.Name
	if serverName == "" {
		serverName = peer.URL
	}
	federated := make([]FederatedAgent, len(agents))
	for i, a := range agents {
		federated[i] = FederatedAgent{Agent: a, Server: serverName}
	}

	// Cache.
	f.mu.Lock()
	f.cache[peer.URL] = &federationCache{agents: federated, fetchedAt: time.Now()}
	f.mu.Unlock()

	return federated, nil
}

// AggregateAgents fetches agents from all peers in parallel, merges with local agents.
func (f *FederationClient) AggregateAgents(ctx context.Context, localAgents []Agent, peers []FederationPeer) []FederatedAgent {
	// Start with local agents.
	result := make([]FederatedAgent, len(localAgents))
	for i, a := range localAgents {
		result[i] = FederatedAgent{Agent: a, Server: "local"}
	}

	if len(peers) == 0 {
		return result
	}

	// Fetch from all peers in parallel.
	type peerResult struct {
		agents []FederatedAgent
		err    error
	}
	ch := make(chan peerResult, len(peers))

	for _, peer := range peers {
		go func(p FederationPeer) {
			agents, err := f.FetchRemoteAgents(ctx, p)
			ch <- peerResult{agents: agents, err: err}
		}(peer)
	}

	for range peers {
		pr := <-ch
		if pr.err != nil {
			continue // Skip unreachable peers.
		}
		result = append(result, pr.agents...)
	}

	return result
}

// CheckPeerHealth verifies a remote peer is reachable.
func (f *FederationClient) CheckPeerHealth(ctx context.Context, peerURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", peerURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("creating health request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("peer unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer health check returned HTTP %d", resp.StatusCode)
	}
	return nil
}
