package controlplane

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// MetricsProxy queries Prometheus for per-agent metric summaries.
type MetricsProxy struct {
	prometheusURL string
	client        *http.Client
	cache         map[string]*cachedMetrics
	mu            sync.RWMutex
	cacheTTL      time.Duration
}

type cachedMetrics struct {
	metrics   *AgentMetrics
	fetchedAt time.Time
}

// AgentMetrics holds summarized metrics for an agent.
type AgentMetrics struct {
	Up         *float64 `json:"up"`
	LatencyP95 *float64 `json:"latency_p95"`
	ErrorRate  *float64 `json:"error_rate"`
}

// NewMetricsProxy creates a metrics proxy targeting the given Prometheus URL.
func NewMetricsProxy(prometheusURL string) *MetricsProxy {
	return &MetricsProxy{
		prometheusURL: prometheusURL,
		client:        &http.Client{Timeout: 5 * time.Second},
		cache:         make(map[string]*cachedMetrics),
		cacheTTL:      30 * time.Second,
	}
}

// GetMetrics returns metric summaries for an agent. Returns nil metrics (not error) if Prometheus is unreachable.
func (m *MetricsProxy) GetMetrics(agentName string) *AgentMetrics {
	// Check cache.
	m.mu.RLock()
	if cached, ok := m.cache[agentName]; ok && time.Since(cached.fetchedAt) < m.cacheTTL {
		m.mu.RUnlock()
		return cached.metrics
	}
	m.mu.RUnlock()

	metrics := &AgentMetrics{}

	// Query each metric independently — partial results are fine.
	if v, err := m.queryScalar(fmt.Sprintf(`up{job="%s"}`, agentName)); err == nil {
		metrics.Up = v
	}
	if v, err := m.queryScalar(fmt.Sprintf(`histogram_quantile(0.95, rate(http_duration_seconds_bucket{job="%s"}[5m]))`, agentName)); err == nil {
		metrics.LatencyP95 = v
	}
	if v, err := m.queryScalar(fmt.Sprintf(`rate(http_requests_total{job="%s",status=~"5.."}[5m])`, agentName)); err == nil {
		metrics.ErrorRate = v
	}

	// Cache result.
	m.mu.Lock()
	m.cache[agentName] = &cachedMetrics{metrics: metrics, fetchedAt: time.Now()}
	m.mu.Unlock()

	return metrics
}

// promResponse represents the Prometheus /api/v1/query response.
type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value []json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// queryScalar executes a PromQL instant query and returns the scalar result.
func (m *MetricsProxy) queryScalar(query string) (*float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", m.prometheusURL, url.QueryEscape(query))
	resp, err := m.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, err
	}

	if pr.Status != "success" || len(pr.Data.Result) == 0 {
		return nil, fmt.Errorf("no result")
	}

	// Result format: [timestamp, "value"]
	if len(pr.Data.Result[0].Value) < 2 {
		return nil, fmt.Errorf("unexpected result format")
	}

	var valStr string
	if err := json.Unmarshal(pr.Data.Result[0].Value[1], &valStr); err != nil {
		return nil, err
	}

	var val float64
	if _, err := fmt.Sscanf(valStr, "%f", &val); err != nil {
		return nil, err
	}

	return &val, nil
}
