package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/romerox3/volra/internal/docker"
)

// MCPProtocolVersion is the protocol version we support.
const MCPProtocolVersion = "2024-11-05"

// Serve runs the MCP server loop, reading JSON-RPC requests from in and
// writing responses to out. Debug messages are logged to stderr.
func Serve(ctx context.Context, in io.Reader, out io.Writer, version string) error {
	runner := docker.NewExecRunner()
	cc := &CallContext{
		Ctx:     ctx,
		Version: version,
		Runner:  runner,
	}

	tools := registry()
	toolMap := make(map[string]ToolDef, len(tools))
	for _, td := range tools {
		toolMap[td.Tool.Name] = td
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max message
	encoder := json.NewEncoder(out)

	log.SetPrefix("[volra-mcp] ")
	log.SetFlags(log.Ltime)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    CodeParseError,
					Message: fmt.Sprintf("Parse error: %s", err),
				},
			}
			_ = encoder.Encode(resp)
			continue
		}

		log.Printf("→ %s (id=%s)", req.Method, string(req.ID))

		resp, isNotification := dispatch(cc, req, tools, toolMap)
		if isNotification {
			continue // notifications don't get responses
		}
		if err := encoder.Encode(resp); err != nil {
			log.Printf("write error: %s", err)
			return err
		}
	}
	return scanner.Err()
}

// dispatch returns the response and whether the request was a notification (no response needed).
func dispatch(cc *CallContext, req Request, tools []ToolDef, toolMap map[string]ToolDef) (Response, bool) {
	switch req.Method {
	case "initialize":
		return handleInitialize(req, cc.Version), false
	case "notifications/initialized":
		return Response{}, true // notifications don't get responses
	case "tools/list":
		return handleToolsList(req, tools), false
	case "tools/call":
		return handleToolsCall(cc, req, toolMap), false
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}, false
	}
}

func handleInitialize(req Request, version string) Response {
	result := InitializeResult{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "volra",
			Version: version,
		},
	}
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func handleToolsList(req Request, tools []ToolDef) Response {
	toolList := make([]Tool, len(tools))
	for i, td := range tools {
		toolList[i] = td.Tool
	}
	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ToolsListResult{Tools: toolList},
	}
}

func handleToolsCall(cc *CallContext, req Request, toolMap map[string]ToolDef) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    CodeInvalidParams,
				Message: fmt.Sprintf("Invalid params: %s", err),
			},
		}
	}

	td, ok := toolMap[params.Name]
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  ErrorResult(fmt.Sprintf("Unknown tool: %s", params.Name)),
		}
	}

	log.Printf("  calling tool: %s", params.Name)
	result := td.Handler(cc, params.Arguments)
	log.Printf("  tool done: %s (error=%v)", params.Name, result.IsError)

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}
