package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/romerox3/volra/internal/mcp"
)

// SessionHeader is the HTTP header for MCP session tracking.
const SessionHeader = "Mcp-Session-Id"

// Server implements the MCP Streamable HTTP transport for the gateway.
type Server struct {
	router   *Router
	sessions *SessionManager
	version  string
	mux      *http.ServeMux
}

// NewServer creates an MCP gateway HTTP server.
func NewServer(router *Router, version string) *Server {
	s := &Server{
		router:   router,
		sessions: NewSessionManager(),
		version:  version,
		mux:      http.NewServeMux(),
	}
	s.mux.HandleFunc("POST /mcp", s.handleMCP)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	toolCount := len(s.router.ListTools())
	resp := map[string]any{
		"status":   "ok",
		"tools":    toolCount,
		"sessions": s.sessions.Count(),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleMCP handles JSON-RPC 2.0 requests over Streamable HTTP.
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB max
	if err != nil {
		writeJSONRPCError(w, nil, mcp.CodeParseError, "Failed to read request body")
		return
	}

	var req mcp.Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, mcp.CodeParseError, fmt.Sprintf("Parse error: %s", err))
		return
	}

	// Session management: read or create session.
	clientSessionID := r.Header.Get(SessionHeader)
	session, isNew := s.sessions.GetOrCreate(clientSessionID)
	w.Header().Set(SessionHeader, session.ID)

	sessionPrefix := session.ID
	if len(sessionPrefix) > 8 {
		sessionPrefix = sessionPrefix[:8]
	}
	log.Printf("[gateway] → %s (id=%s, session=%s, new=%v)", req.Method, string(req.ID), sessionPrefix, isNew)

	resp, isNotification := s.dispatch(r.Context(), req, session)
	if isNotification {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[gateway] write error: %s", err)
	}
}

// dispatch routes the JSON-RPC request to the appropriate handler.
// Returns the response and whether the request was a notification (no response needed).
func (s *Server) dispatch(ctx context.Context, req mcp.Request, session *Session) (mcp.Response, bool) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req), false
	case "notifications/initialized":
		return mcp.Response{}, true
	case "tools/list":
		return s.handleToolsList(req), false
	case "tools/call":
		return s.handleToolsCall(ctx, req, session), false
	default:
		return mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &mcp.RPCError{
				Code:    mcp.CodeMethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}, false
	}
}

func (s *Server) handleInitialize(req mcp.Request) mcp.Response {
	return mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcp.InitializeResult{
			ProtocolVersion: mcp.MCPProtocolVersion,
			Capabilities: mcp.ServerCapabilities{
				Tools: &mcp.ToolsCapability{ListChanged: true},
			},
			ServerInfo: mcp.ServerInfo{
				Name:    "volra-gateway",
				Version: s.version,
			},
		},
	}
}

func (s *Server) handleToolsList(req mcp.Request) mcp.Response {
	return mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  mcp.ToolsListResult{Tools: s.router.ListTools()},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req mcp.Request, session *Session) mcp.Response {
	var params mcp.ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &mcp.RPCError{
				Code:    mcp.CodeInvalidParams,
				Message: fmt.Sprintf("Invalid params: %s", err),
			},
		}
	}

	// Record session-agent interaction for affinity tracking.
	// For three-tier names (server/agent/tool), extract the agent component.
	if nt, ok := s.router.catalog.Lookup(params.Name); ok {
		s.sessions.RecordInteraction(session.ID, nt.AgentName)
	} else if agentName, _, ok := ParseNamespace(params.Name); ok {
		s.sessions.RecordInteraction(session.ID, agentName)
	}

	result, err := s.router.Dispatch(ctx, params.Name, params.Arguments)
	if err != nil {
		return mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  mcp.ErrorResult(fmt.Sprintf("Backend error: %s", err)),
		}
	}

	return mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// writeJSONRPCError writes a JSON-RPC error response.
func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	resp := mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &mcp.RPCError{Code: code, Message: msg},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
