package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/romerox3/volra/internal/output"
)

// DefaultTimeout is the default timeout for A2A card fetch requests.
const DefaultTimeout = 5 * time.Second

// CardResult holds the result of fetching a single agent's card.
type CardResult struct {
	URL   string
	Card  *AgentCard
	Error error
}

// FetchCard fetches an A2A agent card from the given base URL.
// It GETs {baseURL}/.well-known/agent-card.json and parses the response.
func FetchCard(ctx context.Context, baseURL string) (*AgentCard, error) {
	client := &http.Client{Timeout: DefaultTimeout}

	url := baseURL + "/.well-known/agent-card.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &output.UserError{
			Code:    output.CodeA2ACardFetchFailed,
			What: fmt.Sprintf("creating request for %s: %v", url, err),
			Fix:     "Check the agent URL is correct.",
		}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, &output.UserError{
			Code:    output.CodeA2ACardFetchFailed,
			What: fmt.Sprintf("fetching agent card from %s: %v", url, err),
			Fix:     "Verify the remote agent is running and reachable.",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &output.UserError{
			Code:    output.CodeA2ACardFetchFailed,
			What: fmt.Sprintf("agent card at %s returned HTTP %d", url, resp.StatusCode),
			Fix:     "Verify the agent serves an A2A card at /.well-known/agent-card.json",
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, &output.UserError{
			Code:    output.CodeA2ACardFetchFailed,
			What: fmt.Sprintf("reading agent card response from %s: %v", url, err),
			Fix:     "Check network connectivity to the remote agent.",
		}
	}

	var card AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, &output.UserError{
			Code:    output.CodeA2ACardInvalid,
			What: fmt.Sprintf("parsing agent card from %s: %v", url, err),
			Fix:     "The remote agent returned invalid JSON. Check its A2A card configuration.",
		}
	}

	return &card, nil
}

// FetchCards concurrently fetches A2A cards from multiple agent URLs.
// It never fails entirely — returns partial results with per-URL errors.
func FetchCards(ctx context.Context, urls []string) []CardResult {
	results := make([]CardResult, len(urls))
	var wg sync.WaitGroup

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			card, err := FetchCard(ctx, url)
			results[idx] = CardResult{
				URL:   url,
				Card:  card,
				Error: err,
			}
		}(i, u)
	}

	wg.Wait()
	return results
}
