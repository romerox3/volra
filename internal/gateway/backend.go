package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/romerox3/volra/internal/mcp"
)

// SubprocessBackend implements Backend by spawning `volra mcp` for each call.
// This is the simplest implementation — each call starts a new subprocess.
// A persistent-process pool can be added later for performance.
type SubprocessBackend struct {
	// Binary is the path to the volra binary. If empty, uses "volra" from PATH.
	Binary string
}

// Call spawns `volra mcp` in the agent's directory and sends a full
// initialize + tools/call handshake.
func (b *SubprocessBackend) Call(ctx context.Context, agentDir string, params mcp.ToolCallParams) (*mcp.ToolCallResult, error) {
	binary := b.Binary
	if binary == "" {
		binary = "volra"
	}

	cmd := exec.CommandContext(ctx, binary, "mcp")
	cmd.Dir = agentDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting volra mcp: %w", err)
	}

	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	encoder := json.NewEncoder(stdin)

	// 1. Initialize.
	initReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"volra-gateway"}}`),
	}
	if err := encoder.Encode(initReq); err != nil {
		return nil, fmt.Errorf("sending initialize: %w", err)
	}

	scanner := newLineScanner(stdout)

	// Read initialize response.
	if _, err := readJSONRPCResponse(ctx, scanner); err != nil {
		return nil, fmt.Errorf("initialize handshake: %w", err)
	}

	// 2. Send initialized notification.
	notif := mcp.Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := encoder.Encode(notif); err != nil {
		return nil, fmt.Errorf("sending initialized: %w", err)
	}

	// 3. Send tools/call.
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encoding tool params: %w", err)
	}
	callReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}
	if err := encoder.Encode(callReq); err != nil {
		return nil, fmt.Errorf("sending tools/call: %w", err)
	}

	// 4. Read tools/call response.
	callResp, err := readJSONRPCResponse(ctx, scanner)
	if err != nil {
		return nil, fmt.Errorf("reading tools/call response: %w", err)
	}
	if callResp.Error != nil {
		return nil, fmt.Errorf("tools/call error: %s", callResp.Error.Message)
	}

	// Parse result.
	resultBytes, err := json.Marshal(callResp.Result)
	if err != nil {
		return nil, fmt.Errorf("encoding call result: %w", err)
	}
	var result mcp.ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("parsing call result: %w", err)
	}

	return &result, nil
}
