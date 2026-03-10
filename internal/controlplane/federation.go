package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/romerox3/volra/internal/a2a"
)

// FederatedAgent is an agent with its origin server.
type FederatedAgent struct {
	Agent
	Server string `json:"server"`
}

// FederatedCapability is an agent with its A2A card and status.
type FederatedCapability struct {
	Server    string        `json:"server"`
	Agent     string        `json:"agent"`
	URL       string        `json:"url"`
	Status    string        `json:"status"` // "ok", "unreachable", "card_error"
	Card      *a2a.AgentCard `json:"card,omitempty"`
	Error     string        `json:"error,omitempty"`
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

// FetchCapabilities fetches agents from all peers and their A2A cards.
// Results are cached for 60 seconds. Unreachable peers or invalid cards
// produce per-agent error entries, not a total failure.
func (f *FederationClient) FetchCapabilities(ctx context.Context, localAgents []Agent, peers []FederationPeer, localURL string) []FederatedCapability {
	var caps []FederatedCapability

	// Local agents.
	for _, ag := range localAgents {
		agentURL := fmt.Sprintf("http://localhost:%d", ag.Port)
		if localURL != "" {
			agentURL = localURL
		}
		card, err := a2a.FetchCard(ctx, agentURL)
		cap := FederatedCapability{
			Server: "local",
			Agent:  ag.Name,
			URL:    agentURL,
		}
		if err != nil {
			cap.Status = "card_error"
			cap.Error = err.Error()
		} else {
			cap.Status = "ok"
			cap.Card = card
		}
		caps = append(caps, cap)
	}

	if len(peers) == 0 {
		return caps
	}

	// Fetch from peers in parallel.
	type peerCaps struct {
		caps []FederatedCapability
	}
	ch := make(chan peerCaps, len(peers))

	for _, peer := range peers {
		go func(p FederationPeer) {
			agents, err := f.FetchRemoteAgents(ctx, p)
			if err != nil {
				ch <- peerCaps{caps: []FederatedCapability{{
					Server: p.Name,
					Agent:  "",
					Status: "unreachable",
					Error:  err.Error(),
				}}}
				return
			}
			var result []FederatedCapability
			for _, ag := range agents {
				agentURL := fmt.Sprintf("http://%s:%d", p.URL, ag.Port)
				// Use the peer's URL host with agent port.
				card, err := a2a.FetchCard(ctx, agentURL)
				cap := FederatedCapability{
					Server: p.Name,
					Agent:  ag.Name,
					URL:    agentURL,
				}
				if err != nil {
					cap.Status = "card_error"
					cap.Error = err.Error()
				} else {
					cap.Status = "ok"
					cap.Card = card
				}
				result = append(result, cap)
			}
			ch <- peerCaps{caps: result}
		}(peer)
	}

	for range peers {
		pr := <-ch
		caps = append(caps, pr.caps...)
	}

	return caps
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
