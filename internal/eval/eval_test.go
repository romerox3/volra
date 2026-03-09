package eval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockQuerier implements MetricsQuerier for testing.
type mockQuerier struct {
	values map[string]float64
	err    error
}

func (m *mockQuerier) Query(_ context.Context, promql string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	v, ok := m.values[promql]
	if !ok {
		return 0, &output.UserError{
			Code: output.CodePromQLFailed,
			What: "no data for: " + promql,
		}
	}
	return v, nil
}

func writeMinimalAgentfile(t *testing.T, dir string, withEval bool) {
	t.Helper()
	content := `version: 1
name: test-agent
framework: generic
port: 8000
health_path: /health
dockerfile: auto
`
	if withEval {
		content += `eval:
  metrics:
    - name: response_time
      query: "histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))"
      threshold: 20
      unit: seconds
    - name: uptime
      query: "avg_over_time(up[1h])"
      threshold: 5
      unit: ratio
  baseline_window: "1h"
`
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Agentfile"), []byte(content), 0o644))
}

func TestRecord_Success(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0.245,
		"avg_over_time(up[1h])": 0.998,
	}}
	mp := &testutil.MockPresenter{}

	err := Record(context.Background(), dir, mp, q)
	require.NoError(t, err)

	// Verify latest baseline file.
	latestPath := filepath.Join(dir, baselinesDir, "test-agent-latest.json")
	assert.FileExists(t, latestPath)

	data, err := os.ReadFile(latestPath)
	require.NoError(t, err)

	var baseline Baseline
	require.NoError(t, json.Unmarshal(data, &baseline))
	assert.Equal(t, "test-agent", baseline.AgentName)
	assert.Equal(t, "1h", baseline.Window)
	assert.InDelta(t, 0.245, baseline.Metrics["response_time"].Value, 0.001)
	assert.InDelta(t, 0.998, baseline.Metrics["uptime"].Value, 0.001)
}

func TestRecord_NoEvalConfig(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, false)

	q := &mockQuerier{}
	mp := &testutil.MockPresenter{}

	err := Record(context.Background(), dir, mp, q)
	require.Error(t, err)

	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodeInvalidEvalConfig, ue.Code)
}

func TestRecord_PrometheusError(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	q := &mockQuerier{err: &output.UserError{
		Code: output.CodePrometheusUnreachable,
		What: "Prometheus not reachable",
	}}
	mp := &testutil.MockPresenter{}

	err := Record(context.Background(), dir, mp, q)
	require.Error(t, err)

	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodePrometheusUnreachable, ue.Code)
}

func TestCompare_AllPass(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	// Write baseline.
	bDir := filepath.Join(dir, baselinesDir)
	require.NoError(t, os.MkdirAll(bDir, 0o755))
	baseline := Baseline{
		AgentName: "test-agent",
		Window:    "1h",
		Metrics: map[string]MetricValue{
			"response_time": {Value: 0.245, Unit: "seconds"},
			"uptime":        {Value: 0.998, Unit: "ratio"},
		},
	}
	data, _ := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(bDir, "test-agent-latest.json"), data, 0o644))

	// Current values within threshold.
	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0.260, // +6% (threshold 20%)
		"avg_over_time(up[1h])": 0.995, // -0.3% (threshold 5%)
	}}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, false)
	require.NoError(t, err)
	assert.Contains(t, mp.ResultCalls[len(mp.ResultCalls)-1], "All metrics within thresholds")
}

func TestCompare_FailsOnRegression(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	bDir := filepath.Join(dir, baselinesDir)
	require.NoError(t, os.MkdirAll(bDir, 0o755))
	baseline := Baseline{
		AgentName: "test-agent",
		Metrics: map[string]MetricValue{
			"response_time": {Value: 0.245, Unit: "seconds"},
			"uptime":        {Value: 0.998, Unit: "ratio"},
		},
	}
	data, _ := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(bDir, "test-agent-latest.json"), data, 0o644))

	// Response time regressed 30% (threshold 20%).
	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0.320,
		"avg_over_time(up[1h])": 0.995,
	}}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, false)
	require.Error(t, err)

	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Contains(t, ue.What, "1/2 metrics exceeded threshold")
}

func TestCompare_ZeroBaselineRegression(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	bDir := filepath.Join(dir, baselinesDir)
	require.NoError(t, os.MkdirAll(bDir, 0o755))
	baseline := Baseline{
		AgentName: "test-agent",
		Metrics: map[string]MetricValue{
			"response_time": {Value: 0, Unit: "seconds"},
			"uptime":        {Value: 0.998, Unit: "ratio"},
		},
	}
	data, _ := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(bDir, "test-agent-latest.json"), data, 0o644))

	// Baseline was 0, current is non-zero — should detect regression.
	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0.500,
		"avg_over_time(up[1h])": 0.995,
	}}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, false)
	require.Error(t, err)

	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodeEvalRegression, ue.Code)
}

func TestCompare_ZeroBothPasses(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	bDir := filepath.Join(dir, baselinesDir)
	require.NoError(t, os.MkdirAll(bDir, 0o755))
	baseline := Baseline{
		AgentName: "test-agent",
		Metrics: map[string]MetricValue{
			"response_time": {Value: 0, Unit: "seconds"},
			"uptime":        {Value: 0.998, Unit: "ratio"},
		},
	}
	data, _ := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(bDir, "test-agent-latest.json"), data, 0o644))

	// Both zero — should pass.
	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0,
		"avg_over_time(up[1h])": 0.995,
	}}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, false)
	require.NoError(t, err)
}

func TestCompare_NoBaseline(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	q := &mockQuerier{}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, false)
	require.Error(t, err)

	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodeNoBaseline, ue.Code)
}

func TestCompare_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeMinimalAgentfile(t, dir, true)

	bDir := filepath.Join(dir, baselinesDir)
	require.NoError(t, os.MkdirAll(bDir, 0o755))
	baseline := Baseline{
		AgentName: "test-agent",
		Metrics: map[string]MetricValue{
			"response_time": {Value: 0.245, Unit: "seconds"},
			"uptime":        {Value: 0.998, Unit: "ratio"},
		},
	}
	data, _ := json.MarshalIndent(baseline, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(bDir, "test-agent-latest.json"), data, 0o644))

	q := &mockQuerier{values: map[string]float64{
		"histogram_quantile(0.95, rate(http_duration_seconds_bucket[5m]))": 0.250,
		"avg_over_time(up[1h])": 0.997,
	}}
	mp := &testutil.MockPresenter{}

	err := Compare(context.Background(), dir, mp, q, true)
	require.NoError(t, err)

	// Verify JSON output.
	var results []EvalResult
	require.NoError(t, json.Unmarshal([]byte(mp.ResultCalls[0]), &results))
	assert.Len(t, results, 2)
}

func TestRun_InvalidSubcommand(t *testing.T) {
	err := Run(context.Background(), ".", "invalid", nil, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown eval subcommand")
}
