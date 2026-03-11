package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/romerox3/volra/internal/a2a"
)

// Handler is the smart sidecar HTTP handler.
// Routes: GET /.well-known/agent-card.json → card
//         POST /a2a → A2A JSON-RPC handler
//         /* → reverse proxy to agent
type Handler struct {
	cardPath string
	caller   *AgentCaller
	reverse  http.Handler
	tasks    map[string]*a2a.Task
	mu       sync.RWMutex
}

// NewHandler creates the proxy handler.
func NewHandler(agentURL, cardPath string, caller *AgentCaller) (*Handler, error) {
	target, err := url.Parse(agentURL)
	if err != nil {
		return nil, fmt.Errorf("parsing agent URL: %w", err)
	}

	h := &Handler{
		cardPath: cardPath,
		caller:   caller,
		reverse:  httputil.NewSingleHostReverseProxy(target),
		tasks:    make(map[string]*a2a.Task),
	}

	go h.cleanup()

	return h, nil
}

// ServeHTTP routes requests to the appropriate handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/.well-known/agent-card.json" && r.Method == http.MethodGet:
		h.serveCard(w, r)
	case r.URL.Path == "/a2a" && r.Method == http.MethodPost:
		h.handleA2A(w, r)
	default:
		h.reverse.ServeHTTP(w, r)
	}
}

func (h *Handler) serveCard(w http.ResponseWriter, _ *http.Request) {
	data, err := os.ReadFile(h.cardPath)
	if err != nil {
		http.Error(w, "agent card not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Handler) handleA2A(w http.ResponseWriter, r *http.Request) {
	var req a2a.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, nil, &a2a.JSONRPCError{Code: -32700, Message: "parse error"})
		return
	}

	switch req.Method {
	case "Tasks/send":
		h.handleTaskSend(w, req)
	case "Tasks/get":
		h.handleTaskGet(w, req)
	case "Tasks/cancel":
		h.handleTaskCancel(w, req)
	default:
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)})
	}
}

func (h *Handler) handleTaskSend(w http.ResponseWriter, req a2a.JSONRPCRequest) {
	var params a2a.TaskSendParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	// Extract text from message.
	var text string
	for _, part := range params.Message.Parts {
		if part.Type == "text" {
			text = part.Text
			break
		}
	}

	task := a2a.NewTask(params.Message)
	h.mu.Lock()
	h.tasks[task.ID] = task
	h.mu.Unlock()

	task.Transition(a2a.TaskStateWorking, nil)

	// Call the agent via the configured mode.
	result, err := h.caller.Call("", text, req.Params)
	if err != nil {
		task.Fail(err.Error())
	} else {
		task.Complete([]a2a.Artifact{
			{Parts: []a2a.Part{{Type: "text", Text: result}}},
		})
	}

	writeRPCResult(w, req.ID, task)
}

func (h *Handler) handleTaskGet(w http.ResponseWriter, req a2a.JSONRPCRequest) {
	var params a2a.TaskGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	h.mu.RLock()
	task, ok := h.tasks[params.ID]
	h.mu.RUnlock()

	if !ok {
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32001, Message: fmt.Sprintf("task %q not found", params.ID)})
		return
	}

	writeRPCResult(w, req.ID, task)
}

func (h *Handler) handleTaskCancel(w http.ResponseWriter, req a2a.JSONRPCRequest) {
	var params a2a.TaskGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	h.mu.Lock()
	task, ok := h.tasks[params.ID]
	if ok && task.Status.State == a2a.TaskStateWorking {
		task.Cancel()
	}
	h.mu.Unlock()

	if !ok {
		writeRPC(w, req.ID, &a2a.JSONRPCError{Code: -32001, Message: fmt.Sprintf("task %q not found", params.ID)})
		return
	}

	writeRPCResult(w, req.ID, task)
}

func (h *Handler) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		for id, task := range h.tasks {
			if task.Status.State == a2a.TaskStateCompleted || task.Status.State == a2a.TaskStateFailed || task.Status.State == a2a.TaskStateCanceled {
				if time.Since(task.UpdatedAt) > 5*time.Minute {
					delete(h.tasks, id)
				}
			}
		}
		h.mu.Unlock()
	}
}

func writeRPC(w http.ResponseWriter, id json.RawMessage, rpcErr *a2a.JSONRPCError) {
	resp := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcErr,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result interface{}) {
	resp := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
