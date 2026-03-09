// Package eval implements the volra eval command logic (baseline recording and regression detection).
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/output"
)

// Baseline represents a recorded metric snapshot.
type Baseline struct {
	RecordedAt time.Time          `json:"recorded_at"`
	AgentName  string             `json:"agent_name"`
	Window     string             `json:"window"`
	Metrics    map[string]MetricValue `json:"metrics"`
}

// MetricValue holds a single metric's recorded value.
type MetricValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// EvalResult holds the result of a single metric comparison.
type EvalResult struct {
	Name      string  `json:"name"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Threshold int     `json:"threshold"`
	Deviation float64 `json:"deviation"`
	Passed    bool    `json:"passed"`
	Unit      string  `json:"unit"`
}

const baselinesDir = ".volra/baselines"

// Run dispatches the eval subcommand.
func Run(ctx context.Context, dir, subcmd string, p output.Presenter, querier MetricsQuerier, jsonOut bool) error {
	switch subcmd {
	case "record":
		return Record(ctx, dir, p, querier)
	case "run":
		return Compare(ctx, dir, p, querier, jsonOut)
	default:
		return fmt.Errorf("unknown eval subcommand: %s (use 'record' or 'run')", subcmd)
	}
}

// Record captures current metrics as a baseline.
func Record(ctx context.Context, dir string, p output.Presenter, querier MetricsQuerier) error {
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	if err != nil {
		return err
	}

	if af.Eval == nil || len(af.Eval.Metrics) == 0 {
		return &output.UserError{
			Code: output.CodeInvalidEvalConfig,
			What: "No eval config in Agentfile",
			Fix:  "Add an 'eval' section with metrics to your Agentfile",
		}
	}

	p.Progress("Recording baseline metrics...")

	baseline := Baseline{
		RecordedAt: time.Now().UTC(),
		AgentName:  af.Name,
		Window:     af.Eval.BaselineWindow,
		Metrics:    make(map[string]MetricValue),
	}
	if baseline.Window == "" {
		baseline.Window = "1h"
	}

	for _, m := range af.Eval.Metrics {
		val, err := querier.Query(ctx, m.Query)
		if err != nil {
			return err
		}
		baseline.Metrics[m.Name] = MetricValue{Value: val, Unit: m.Unit}
		p.Progress(fmt.Sprintf("  %s = %.4f %s", m.Name, val, m.Unit))
	}

	// Save baseline.
	bDir := filepath.Join(dir, baselinesDir)
	if err := os.MkdirAll(bDir, 0o755); err != nil {
		return fmt.Errorf("creating baselines directory: %w", err)
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding baseline: %w", err)
	}

	// Write timestamped file.
	ts := baseline.RecordedAt.Format("20060102-150405")
	tsPath := filepath.Join(bDir, fmt.Sprintf("%s-%s.json", af.Name, ts))
	if err := os.WriteFile(tsPath, data, 0o644); err != nil {
		return fmt.Errorf("writing baseline: %w", err)
	}

	// Write latest file.
	latestPath := filepath.Join(bDir, fmt.Sprintf("%s-latest.json", af.Name))
	if err := os.WriteFile(latestPath, data, 0o644); err != nil {
		return fmt.Errorf("writing latest baseline: %w", err)
	}

	p.Result(fmt.Sprintf("Baseline recorded: %s", tsPath))
	return nil
}

// Compare runs current metrics against the latest baseline.
func Compare(ctx context.Context, dir string, p output.Presenter, querier MetricsQuerier, jsonOut bool) error {
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	if err != nil {
		return err
	}

	if af.Eval == nil || len(af.Eval.Metrics) == 0 {
		return &output.UserError{
			Code: output.CodeInvalidEvalConfig,
			What: "No eval config in Agentfile",
			Fix:  "Add an 'eval' section with metrics to your Agentfile",
		}
	}

	// Load latest baseline.
	latestPath := filepath.Join(dir, baselinesDir, fmt.Sprintf("%s-latest.json", af.Name))
	baselineData, err := os.ReadFile(latestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &output.UserError{
				Code: output.CodeNoBaseline,
				What: "No baseline found",
				Fix:  "Run `volra eval record` first to capture a baseline",
			}
		}
		return fmt.Errorf("reading baseline: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		return fmt.Errorf("parsing baseline: %w", err)
	}

	p.Progress("Comparing current metrics against baseline...")

	var results []EvalResult
	anyFailed := false

	for _, m := range af.Eval.Metrics {
		bv, ok := baseline.Metrics[m.Name]
		if !ok {
			p.Warn(&output.UserWarning{
				What:    fmt.Sprintf("Metric %q not found in baseline — skipping", m.Name),
				Assumed: "This metric was added after the baseline was recorded",
			})
			continue
		}

		current, err := querier.Query(ctx, m.Query)
		if err != nil {
			return err
		}

		var deviation float64
		if bv.Value == 0 {
			if current != 0 {
				deviation = math.Inf(1)
			}
		} else {
			deviation = math.Abs(current-bv.Value) / math.Abs(bv.Value) * 100
		}

		passed := deviation <= float64(m.Threshold)
		if !passed {
			anyFailed = true
		}

		results = append(results, EvalResult{
			Name:      m.Name,
			Baseline:  bv.Value,
			Current:   current,
			Threshold: m.Threshold,
			Deviation: deviation,
			Passed:    passed,
			Unit:      m.Unit,
		})
	}

	if jsonOut {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding eval results: %w", err)
		}
		p.Result(string(data))
	} else {
		printSummaryTable(p, results)
	}

	if anyFailed {
		failCount := 0
		for _, r := range results {
			if !r.Passed {
				failCount++
			}
		}
		return &output.UserError{
			Code: output.CodeEvalRegression,
			What: fmt.Sprintf("Eval failed: %d/%d metrics exceeded threshold", failCount, len(results)),
			Fix:  "Investigate the regressed metrics or update the baseline with `volra eval record`",
		}
	}

	p.Result("All metrics within thresholds")
	return nil
}

func printSummaryTable(p output.Presenter, results []EvalResult) {
	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "Metric\tBaseline\tCurrent\tThreshold\tStatus")
	for _, r := range results {
		status := "✓ PASS"
		if !r.Passed {
			if math.IsInf(r.Deviation, 1) {
				status = "✗ FAIL (∞)"
			} else {
				status = fmt.Sprintf("✗ FAIL (+%.1f%%)", r.Deviation)
			}
		} else {
			sign := "+"
			if r.Current < r.Baseline {
				sign = ""
			}
			status = fmt.Sprintf("✓ PASS (%s%.1f%%)", sign, r.Deviation)
		}

		fmt.Fprintf(w, "%s\t%.4f %s\t%.4f %s\t%d%%\t%s\n",
			r.Name, r.Baseline, r.Unit, r.Current, r.Unit, r.Threshold, status)
	}
	w.Flush()

	p.Result(string(buf))
}

// writerFunc adapts a function to io.Writer.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }
