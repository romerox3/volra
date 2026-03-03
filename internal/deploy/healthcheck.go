package deploy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/antonioromero/volra/internal/output"
)

const (
	healthRetryInterval = 2 * time.Second
	healthTimeout       = 60 * time.Second
)

// WaitForHealth polls the agent's health endpoint until it responds with HTTP 200.
func WaitForHealth(ctx context.Context, port int, healthPath string, name string, p output.Presenter) error {
	url := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	deadline := time.Now().Add(healthTimeout)

	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	p.Progress("Waiting for agent health...")

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("creating health request: %w", err)
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if time.Now().After(deadline) || ctx.Err() != nil {
			return &output.UserError{
				Code: output.CodeHealthCheckFailed,
				What: fmt.Sprintf("Health check timed out after %s — %s did not respond", healthTimeout, url),
				Fix:  fmt.Sprintf("Check agent logs: docker logs %s-agent", name),
			}
		}

		select {
		case <-ctx.Done():
			return &output.UserError{
				Code: output.CodeHealthCheckFailed,
				What: fmt.Sprintf("Health check timed out after %s — %s did not respond", healthTimeout, url),
				Fix:  fmt.Sprintf("Check agent logs: docker logs %s-agent", name),
			}
		case <-time.After(healthRetryInterval):
			p.Progress("Waiting for agent health...")
		}
	}
}
