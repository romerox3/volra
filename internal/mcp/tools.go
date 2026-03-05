package mcp

import "encoding/json"

// ToolDef holds a tool's static definition and its handler function.
type ToolDef struct {
	Tool    Tool
	Handler func(ctx *CallContext, args json.RawMessage) *ToolCallResult
}

// registry returns all tools registered for the MCP server.
func registry() []ToolDef {
	return []ToolDef{
		{
			Tool: Tool{
				Name:        "volra_deploy",
				Description: "Deploy an AI agent from a project directory. Generates Docker Compose, Prometheus, and Grafana configs, then starts all services.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the agent project directory (default: current working directory)",
						},
					},
				},
			},
			Handler: handleDeploy,
		},
		{
			Tool: Tool{
				Name:        "volra_status",
				Description: "Check the health and status of a deployed agent and its services.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the agent project directory (default: current working directory)",
						},
					},
				},
			},
			Handler: handleStatus,
		},
		{
			Tool: Tool{
				Name:        "volra_logs",
				Description: "Get recent logs from a deployed agent or a specific service.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the agent project directory (default: current working directory)",
						},
						"lines": map[string]any{
							"type":        "integer",
							"description": "Number of log lines to return (default: 50)",
						},
						"service": map[string]any{
							"type":        "string",
							"description": "Specific service name to get logs from (default: agent container)",
						},
					},
				},
			},
			Handler: handleLogs,
		},
		{
			Tool: Tool{
				Name:        "volra_doctor",
				Description: "Run diagnostic checks on system prerequisites (Docker, Compose, Python, disk space, ports).",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{},
				},
			},
			Handler: handleDoctor,
		},
	}
}
