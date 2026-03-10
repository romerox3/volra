package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/romerox3/volra/internal/output"
)

// A2ABackend sends tool calls to remote agents via A2A JSON-RPC (Tasks/send).
type A2ABackend struct {
	client *http.Client
}

// NewA2ABackend creates a new A2A backend with a 30s timeout.
func NewA2ABackend() *A2ABackend {
	return &A2ABackend{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// a2aRequest is a JSON-RPC 2.0 request for A2A.
type a2aRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// a2aTaskSendParams are the params for Tasks/send.
type a2aTaskSendParams struct {
	Message a2aMessage `json:"message"`
}

// a2aMessage is a message in the A2A protocol.
type a2aMessage struct {
	Role  string    `json:"role"`
	Parts []a2aPart `json:"parts"`
}

// a2aPart is a content part in an A2A message.
type a2aPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// a2aResponse is a JSON-RPC 2.0 response.
type a2aResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  *a2aTaskResult   `json:"result,omitempty"`
	Error   *a2aResponseError `json:"error,omitempty"`
}

// a2aResponseError is a JSON-RPC error.
type a2aResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// a2aTaskResult is the result of a Tasks/send call.
type a2aTaskResult struct {
	ID        string         `json:"id"`
	Status    a2aTaskStatus  `json:"status"`
	Artifacts []a2aArtifact  `json:"artifacts,omitempty"`
}

// a2aTaskStatus is the status of an A2A task.
type a2aTaskStatus struct {
	State string `json:"state"`
}

// a2aArtifact is an artifact returned by a task.
type a2aArtifact struct {
	Parts []a2aPart `json:"parts"`
}

// CallRemote sends a tool call to a remote agent via A2A Tasks/send.
func (b *A2ABackend) CallRemote(ctx context.Context, agentURL string, toolName string, arguments json.RawMessage) (*mcp.ToolCallResult, error) {
	// Build the text payload: tool name + arguments.
	payload := fmt.Sprintf("Call tool %q with arguments: %s", toolName, string(arguments))

	reqBody := a2aRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "Tasks/send",
		Params: a2aTaskSendParams{
			Message: a2aMessage{
				Role: "user",
				Parts: []a2aPart{
					{Type: "text", Text: payload},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling A2A request: %w", err)
	}

	url := agentURL + "/a2a"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &output.UserError{
			Code: output.CodeA2ARemoteCallFailed,
			What: fmt.Sprintf("creating request to %s: %v", url, err),
			Fix:  "Check the remote agent URL.",
		}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, &output.UserError{
			Code: output.CodeA2ARemoteCallFailed,
			What: fmt.Sprintf("calling remote agent at %s: %v", url, err),
			Fix:  "Verify the remote agent is running and reachable.",
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading A2A response: %w", err)
	}

	var a2aResp a2aResponse
	if err := json.Unmarshal(respBody, &a2aResp); err != nil {
		return nil, &output.UserError{
			Code: output.CodeA2ARemoteCallFailed,
			What: fmt.Sprintf("parsing A2A response from %s: %v", url, err),
			Fix:  "The remote agent returned invalid JSON-RPC.",
		}
	}

	if a2aResp.Error != nil {
		return nil, &output.UserError{
			Code: output.CodeA2ARemoteCallFailed,
			What: fmt.Sprintf("remote agent error: %s", a2aResp.Error.Message),
			Fix:  "Check the remote agent logs for details.",
		}
	}

	if a2aResp.Result == nil {
		return mcp.ErrorResult("remote agent returned empty result"), nil
	}

	if a2aResp.Result.Status.State == "failed" {
		return mcp.ErrorResult("remote task failed"), nil
	}

	// Extract text from first artifact.
	if len(a2aResp.Result.Artifacts) > 0 && len(a2aResp.Result.Artifacts[0].Parts) > 0 {
		text := a2aResp.Result.Artifacts[0].Parts[0].Text
		return mcp.SuccessResult(text), nil
	}

	return mcp.SuccessResult("task completed"), nil
}
