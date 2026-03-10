// Package gateway implements the MCP Gateway for multi-agent tool routing.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/romerox3/volra/internal/mcp"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
)

// DefaultDiscoveryTimeout is the maximum time to wait for a single agent's tools/list response.
const DefaultDiscoveryTimeout = 5 * time.Second

// NamespacedTool wraps an MCP tool with its owning agent name.
type NamespacedTool struct {
	AgentName    string   `json:"agent_name"`
	OriginalName string   `json:"original_name"`
	Tool         mcp.Tool `json:"tool"`
	// Remote fields (empty for local tools).
	Server   string `json:"server,omitempty"`   // Federation server name.
	AgentURL string `json:"agent_url,omitempty"` // Remote agent base URL.
	Remote   bool   `json:"remote,omitempty"`    // True for federated tools.
}

// Catalog holds the unified tool catalog across all agents.
type Catalog struct {
	mu    sync.RWMutex
	tools []NamespacedTool
}

// Tools returns a snapshot of all namespaced tools.
func (c *Catalog) Tools() []NamespacedTool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make([]NamespacedTool, len(c.tools))
	copy(cp, c.tools)
	return cp
}

// Lookup finds a tool by its namespaced name (agent-name/tool-name).
func (c *Catalog) Lookup(namespacedName string) (NamespacedTool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, t := range c.tools {
		if t.Tool.Name == namespacedName {
			return t, true
		}
	}
	return NamespacedTool{}, false
}

// MCPSpawner abstracts spawning an MCP subprocess for an agent.
// This allows testing without real subprocesses.
type MCPSpawner interface {
	// DiscoverTools spawns an MCP server for the given agent directory,
	// performs the initialize handshake, and returns the tool list.
	DiscoverTools(ctx context.Context, agentDir string) ([]mcp.Tool, error)
}

// SubprocessSpawner discovers tools by running `volra mcp` as a subprocess.
type SubprocessSpawner struct {
	// Binary is the path to the volra binary. If empty, uses "volra" from PATH.
	Binary string
}

// DiscoverTools spawns `volra mcp` in the agent's directory, sends initialize + tools/list,
// and returns the discovered tools.
func (s *SubprocessSpawner) DiscoverTools(ctx context.Context, agentDir string) ([]mcp.Tool, error) {
	binary := s.Binary
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
	scanner := newLineScanner(stdout)

	// 1. Initialize handshake.
	initReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"volra-gateway"}}`),
	}
	if err := encoder.Encode(initReq); err != nil {
		return nil, fmt.Errorf("sending initialize: %w", err)
	}

	initResp, err := readJSONRPCResponse(ctx, scanner)
	if err != nil {
		return nil, fmt.Errorf("reading initialize response: %w", err)
	}
	if initResp.Error != nil {
		return nil, fmt.Errorf("initialize error: %s", initResp.Error.Message)
	}

	// 2. Send initialized notification.
	notif := mcp.Request{JSONRPC: "2.0", Method: "notifications/initialized"}
	if err := encoder.Encode(notif); err != nil {
		return nil, fmt.Errorf("sending initialized notification: %w", err)
	}

	// 3. Request tools/list.
	toolsReq := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}
	if err := encoder.Encode(toolsReq); err != nil {
		return nil, fmt.Errorf("sending tools/list: %w", err)
	}

	toolsResp, err := readJSONRPCResponse(ctx, scanner)
	if err != nil {
		return nil, fmt.Errorf("reading tools/list response: %w", err)
	}
	if toolsResp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", toolsResp.Error.Message)
	}

	// Parse the result.
	resultBytes, err := json.Marshal(toolsResp.Result)
	if err != nil {
		return nil, fmt.Errorf("encoding tools result: %w", err)
	}
	var toolsResult mcp.ToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		return nil, fmt.Errorf("parsing tools/list result: %w", err)
	}

	return toolsResult.Tools, nil
}

// BuildCatalog discovers tools from all registered agents and returns a unified catalog.
// Each tool is namespaced as "agent-name/original-tool-name".
// Agents that fail discovery are skipped with a warning.
func BuildCatalog(ctx context.Context, agents []registry.AgentEntry, spawner MCPSpawner, p output.Presenter, timeout time.Duration) (*Catalog, error) {
	if len(agents) == 0 {
		return nil, &output.UserError{
			Code: output.CodeGatewayNoAgents,
			What: "No agents registered",
			Fix:  "Deploy at least one agent with `volra deploy` before starting the gateway",
		}
	}

	if timeout == 0 {
		timeout = DefaultDiscoveryTimeout
	}

	var allTools []NamespacedTool
	discovered := 0

	for _, agent := range agents {
		agentCtx, cancel := context.WithTimeout(ctx, timeout)
		tools, err := spawner.DiscoverTools(agentCtx, agent.ProjectDir)
		cancel()

		if err != nil {
			p.Warn(&output.UserWarning{
				What:    fmt.Sprintf("Agent %q: tool discovery failed: %s", agent.Name, err),
				Assumed: "Agent skipped — its tools will not be available in the gateway",
			})
			continue
		}

		for _, t := range tools {
			nt := NamespacedTool{
				AgentName:    agent.Name,
				OriginalName: t.Name,
				Tool: mcp.Tool{
					Name:        agent.Name + "/" + t.Name,
					Description: fmt.Sprintf("[%s] %s", agent.Name, t.Description),
					InputSchema: t.InputSchema,
				},
			}
			allTools = append(allTools, nt)
		}
		discovered++
		p.Progress(fmt.Sprintf("Discovered %d tools from agent %q", len(tools), agent.Name))
	}

	if discovered == 0 {
		return nil, &output.UserError{
			Code: output.CodeGatewayToolsFailed,
			What: "No tools discovered from any agent",
			Fix:  "Ensure agents are running and `volra mcp` works in their directories",
		}
	}

	return &Catalog{tools: allTools}, nil
}

// AddRemoteTools merges federated tools into the catalog.
// Remote tools use server/agent/tool namespacing.
func (c *Catalog) AddRemoteTools(tools []NamespacedTool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Remove existing remote tools first (for refresh).
	var local []NamespacedTool
	for _, t := range c.tools {
		if !t.Remote {
			local = append(local, t)
		}
	}
	c.tools = append(local, tools...)
}

// RemoteToolCount returns the number of remote tools in the catalog.
func (c *Catalog) RemoteToolCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	for _, t := range c.tools {
		if t.Remote {
			count++
		}
	}
	return count
}
