package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/romerox3/volra/internal/mcp"
)

// Backend abstracts sending a tools/call to a specific agent's MCP server.
type Backend interface {
	// Call sends a tools/call request to the agent and returns the result.
	Call(ctx context.Context, agentDir string, params mcp.ToolCallParams) (*mcp.ToolCallResult, error)
}

// Router dispatches namespaced tool calls to the correct agent backend.
type Router struct {
	catalog *Catalog
	backend Backend

	mu       sync.RWMutex
	agentDir map[string]string // agent name -> project dir
}

// NewRouter creates a router with the given catalog and backend.
// agentDirs maps agent names to their project directories.
func NewRouter(catalog *Catalog, backend Backend, agentDirs map[string]string) *Router {
	return &Router{
		catalog:  catalog,
		backend:  backend,
		agentDir: agentDirs,
	}
}

// Dispatch handles a namespaced tool call (e.g. "agent-a/volra_deploy").
// It resolves the agent, strips the namespace prefix, and forwards to the backend.
func (r *Router) Dispatch(ctx context.Context, namespacedName string, arguments json.RawMessage) (*mcp.ToolCallResult, error) {
	// 1. Lookup in catalog (read lock protects catalog pointer during reload).
	r.mu.RLock()
	nt, ok := r.catalog.Lookup(namespacedName)
	r.mu.RUnlock()
	if !ok {
		return mcp.ErrorResult(fmt.Sprintf("Unknown tool: %s", namespacedName)), nil
	}

	// 2. Resolve agent directory.
	r.mu.RLock()
	dir, ok := r.agentDir[nt.AgentName]
	r.mu.RUnlock()
	if !ok {
		return mcp.ErrorResult(fmt.Sprintf("Agent %q not registered", nt.AgentName)), nil
	}

	// 3. Forward with original (un-namespaced) tool name.
	params := mcp.ToolCallParams{
		Name:      nt.OriginalName,
		Arguments: arguments,
	}

	return r.backend.Call(ctx, dir, params)
}

// ListTools returns the unified tool list for tools/list responses.
func (r *Router) ListTools() []mcp.Tool {
	r.mu.RLock()
	cat := r.catalog
	r.mu.RUnlock()
	nts := cat.Tools()
	tools := make([]mcp.Tool, len(nts))
	for i, nt := range nts {
		tools[i] = nt.Tool
	}
	return tools
}

// ReloadCatalog replaces the router's catalog with a new one.
func (r *Router) ReloadCatalog(catalog *Catalog, agentDirs map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalog = catalog
	r.agentDir = agentDirs
}

// ParseNamespace splits a namespaced tool name into agent name and tool name.
// Returns ("", "", false) if the name has no namespace separator.
func ParseNamespace(namespacedName string) (agentName, toolName string, ok bool) {
	idx := strings.IndexByte(namespacedName, '/')
	if idx <= 0 || idx >= len(namespacedName)-1 {
		return "", "", false
	}
	return namespacedName[:idx], namespacedName[idx+1:], true
}

// ParseThreeTierNamespace splits a three-tier namespace (server/agent/tool).
// Returns ("", "", "", false) if the name doesn't have exactly 3 parts.
func ParseThreeTierNamespace(namespacedName string) (server, agent, tool string, ok bool) {
	parts := strings.SplitN(namespacedName, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// IsRemoteNamespace returns true if the name has 3 parts (server/agent/tool).
func IsRemoteNamespace(name string) bool {
	return strings.Count(name, "/") >= 2
}
