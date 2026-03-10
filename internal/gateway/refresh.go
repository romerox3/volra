package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/romerox3/volra/internal/a2a"
	"github.com/romerox3/volra/internal/controlplane"
	"github.com/romerox3/volra/internal/mcp"
)

// DefaultRefreshInterval is the default interval for federated catalog refresh.
const DefaultRefreshInterval = 60 * time.Second

// RefreshConfig configures the federated catalog refresh.
type RefreshConfig struct {
	ControlPlaneURL string
	Interval        time.Duration
}

// GetRefreshInterval returns the configured interval or checks VOLRA_GATEWAY_REFRESH_INTERVAL.
func GetRefreshInterval() time.Duration {
	if v := os.Getenv("VOLRA_GATEWAY_REFRESH_INTERVAL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return DefaultRefreshInterval
}

// StartRefresh starts a background goroutine that periodically refreshes
// the federated tool catalog from the control plane.
func StartRefresh(ctx context.Context, catalog *Catalog, config RefreshConfig) {
	if config.Interval == 0 {
		config.Interval = GetRefreshInterval()
	}
	if config.ControlPlaneURL == "" {
		return // No control plane configured.
	}

	go func() {
		ticker := time.NewTicker(config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshFederatedTools(ctx, catalog, config.ControlPlaneURL)
			}
		}
	}()
}

// refreshFederatedTools fetches capabilities from the control plane
// and updates the catalog with remote tools.
func refreshFederatedTools(ctx context.Context, catalog *Catalog, controlPlaneURL string) {
	caps, err := fetchCapabilities(ctx, controlPlaneURL)
	if err != nil {
		log.Printf("[gateway] federation refresh failed: %s (keeping stale catalog)", err)
		return
	}

	var remote []NamespacedTool
	for _, cap := range caps {
		if cap.Status != "ok" || cap.Card == nil || cap.Server == "local" {
			continue
		}
		// Create tools from A2A card skills.
		for _, skill := range cap.Card.Skills {
			nt := NamespacedTool{
				AgentName:    cap.Agent,
				OriginalName: skill.ID,
				Server:       cap.Server,
				AgentURL:     cap.URL,
				Remote:       true,
				Tool: mcp.Tool{
					Name:        cap.Server + "/" + cap.Agent + "/" + skill.ID,
					Description: fmt.Sprintf("[%s/%s] %s", cap.Server, cap.Agent, skill.Description),
					InputSchema: map[string]any{"type": "object"},
				},
			}
			remote = append(remote, nt)
		}
	}

	oldCount := catalog.RemoteToolCount()
	catalog.AddRemoteTools(remote)
	newCount := catalog.RemoteToolCount()

	if newCount != oldCount {
		log.Printf("[gateway] federation refresh: %d remote tools (was %d)", newCount, oldCount)
	}
}

// fetchCapabilities calls GET /api/federation/capabilities on the control plane.
func fetchCapabilities(ctx context.Context, controlPlaneURL string) ([]controlplane.FederatedCapability, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, controlPlaneURL+"/api/federation/capabilities", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying control plane: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var caps []controlplane.FederatedCapability
	if err := json.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("parsing capabilities: %w", err)
	}

	return caps, nil
}

// LoadFederatedTools does an initial load of federated tools from the control plane.
// Returns nil on error (graceful degradation — gateway works with local tools only).
func LoadFederatedTools(ctx context.Context, controlPlaneURL string) []NamespacedTool {
	if controlPlaneURL == "" {
		return nil
	}

	caps, err := fetchCapabilities(ctx, controlPlaneURL)
	if err != nil {
		log.Printf("[gateway] warning: federation unavailable: %s (E2004)", err)
		return nil
	}

	var remote []NamespacedTool
	for _, cap := range caps {
		if cap.Status != "ok" || cap.Card == nil || cap.Server == "local" {
			continue
		}
		for _, skill := range cap.Card.Skills {
			nt := NamespacedTool{
				AgentName:    cap.Agent,
				OriginalName: skill.ID,
				Server:       cap.Server,
				AgentURL:     cap.URL,
				Remote:       true,
				Tool: mcp.Tool{
					Name:        cap.Server + "/" + cap.Agent + "/" + skill.ID,
					Description: fmt.Sprintf("[%s/%s] %s", cap.Server, cap.Agent, skill.Description),
					InputSchema: map[string]any{"type": "object"},
				},
			}
			remote = append(remote, nt)
		}
	}

	return remote
}

// Ensure a2a import is used (for caps.Card type).
var _ = a2a.AgentCard{}
