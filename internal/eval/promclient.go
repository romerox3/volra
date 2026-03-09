package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/romerox3/volra/internal/output"
)

// MetricsQuerier queries Prometheus for metric values.
type MetricsQuerier interface {
	Query(ctx context.Context, promql string) (float64, error)
}

// PromClient queries Prometheus via HTTP API.
type PromClient struct {
	BaseURL string
	Client  *http.Client
}

// promResponse represents the Prometheus instant query API response.
type promResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value [2]json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Query executes a PromQL instant query and returns the scalar result.
func (c *PromClient) Query(ctx context.Context, promql string) (float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", c.BaseURL, url.QueryEscape(promql))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, &output.UserError{
			Code: output.CodePrometheusUnreachable,
			What: fmt.Sprintf("Prometheus not reachable at %s", c.BaseURL),
			Fix:  "Is the agent deployed? Check that Prometheus is running.",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}

	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return 0, fmt.Errorf("parsing Prometheus response: %w", err)
	}

	if pr.Status != "success" {
		return 0, &output.UserError{
			Code: output.CodePromQLFailed,
			What: fmt.Sprintf("PromQL query failed: %s — %s", promql, pr.Error),
			Fix:  "Check your PromQL query syntax in the eval config",
		}
	}

	if len(pr.Data.Result) == 0 {
		return 0, &output.UserError{
			Code: output.CodePromQLFailed,
			What: fmt.Sprintf("PromQL query returned no data: %s", promql),
			Fix:  "Verify the metric exists in Prometheus and the agent is reporting data",
		}
	}

	// Value is [timestamp, "value_string"].
	var valStr string
	if err := json.Unmarshal(pr.Data.Result[0].Value[1], &valStr); err != nil {
		return 0, fmt.Errorf("parsing metric value: %w", err)
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("converting metric value %q to float: %w", valStr, err)
	}

	return val, nil
}
