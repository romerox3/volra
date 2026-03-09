package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsProxy_GetMetrics_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := promResponse{Status: "success"}
		resp.Data.ResultType = "vector"
		ts, _ := json.Marshal(1234567890.0)
		val, _ := json.Marshal("1.5")
		resp.Data.Result = []struct {
			Value []json.RawMessage `json:"value"`
		}{{Value: []json.RawMessage{ts, val}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	proxy := NewMetricsProxy(srv.URL)
	metrics := proxy.GetMetrics("test-agent")

	require.NotNil(t, metrics)
	require.NotNil(t, metrics.Up)
	assert.Equal(t, 1.5, *metrics.Up)
}

func TestMetricsProxy_GetMetrics_PrometheusDown(t *testing.T) {
	proxy := NewMetricsProxy("http://localhost:1") // unreachable
	metrics := proxy.GetMetrics("test-agent")

	// Should return metrics with nil fields, not panic.
	require.NotNil(t, metrics)
	assert.Nil(t, metrics.Up)
	assert.Nil(t, metrics.LatencyP95)
	assert.Nil(t, metrics.ErrorRate)
}

func TestMetricsProxy_GetMetrics_UsesCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := promResponse{Status: "success"}
		resp.Data.ResultType = "vector"
		ts, _ := json.Marshal(1234567890.0)
		val, _ := json.Marshal("1.0")
		resp.Data.Result = []struct {
			Value []json.RawMessage `json:"value"`
		}{{Value: []json.RawMessage{ts, val}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	proxy := NewMetricsProxy(srv.URL)

	// First call hits Prometheus.
	proxy.GetMetrics("cached-agent")
	firstCount := callCount

	// Second call should use cache.
	proxy.GetMetrics("cached-agent")
	assert.Equal(t, firstCount, callCount, "second call should use cache")
}

func TestMetricsProxy_GetMetrics_NoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := promResponse{Status: "success"}
		resp.Data.ResultType = "vector"
		// Empty result set.
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	proxy := NewMetricsProxy(srv.URL)
	metrics := proxy.GetMetrics("empty-agent")

	require.NotNil(t, metrics)
	assert.Nil(t, metrics.Up)
}
