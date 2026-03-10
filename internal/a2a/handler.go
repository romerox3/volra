package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TaskSendParams are the params for Tasks/send.
type TaskSendParams struct {
	ID      string  `json:"id,omitempty"`
	Message Message `json:"message"`
}

// TaskGetParams are the params for Tasks/get.
type TaskGetParams struct {
	ID string `json:"id"`
}

// ToolCaller is the interface for calling local tools.
type ToolCaller interface {
	CallTool(toolName string, input string) (string, error)
}

// TaskHandler handles incoming A2A JSON-RPC requests.
type TaskHandler struct {
	tasks  map[string]*Task
	mu     sync.RWMutex
	caller ToolCaller
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(caller ToolCaller) *TaskHandler {
	h := &TaskHandler{
		tasks:  make(map[string]*Task),
		caller: caller,
	}
	// Start cleanup goroutine.
	go h.cleanup()
	return h
}

// ServeHTTP handles POST /a2a requests.
func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONRPC(w, nil, &JSONRPCError{Code: -32600, Message: "method not allowed"})
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPC(w, nil, &JSONRPCError{Code: -32700, Message: "parse error"})
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
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)})
	}
}

func (h *TaskHandler) handleTaskSend(w http.ResponseWriter, req JSONRPCRequest) {
	var params TaskSendParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	task := NewTask(params.Message)
	h.mu.Lock()
	h.tasks[task.ID] = task
	h.mu.Unlock()

	// Extract text from message to use as tool input.
	var input string
	for _, part := range params.Message.Parts {
		if part.Type == "text" {
			input = part.Text
			break
		}
	}

	task.Transition(TaskStateWorking, nil)

	// Call the tool if caller is available.
	if h.caller != nil {
		result, err := h.caller.CallTool("", input)
		if err != nil {
			task.Fail(err.Error())
		} else {
			task.Complete([]Artifact{
				{Parts: []Part{{Type: "text", Text: result}}},
			})
		}
	} else {
		task.Complete([]Artifact{
			{Parts: []Part{{Type: "text", Text: "no tool caller configured"}}},
		})
	}

	writeJSONRPCWithID(w, req.ID, task, nil)
}

func (h *TaskHandler) handleTaskGet(w http.ResponseWriter, req JSONRPCRequest) {
	var params TaskGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	h.mu.RLock()
	task, ok := h.tasks[params.ID]
	h.mu.RUnlock()

	if !ok {
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32001, Message: fmt.Sprintf("task %q not found", params.ID)})
		return
	}

	writeJSONRPCWithID(w, req.ID, task, nil)
}

func (h *TaskHandler) handleTaskCancel(w http.ResponseWriter, req JSONRPCRequest) {
	var params TaskGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32602, Message: "invalid params"})
		return
	}

	h.mu.Lock()
	task, ok := h.tasks[params.ID]
	if ok && task.Status.State == TaskStateWorking {
		task.Cancel()
	}
	h.mu.Unlock()

	if !ok {
		writeJSONRPCWithID(w, req.ID, nil, &JSONRPCError{Code: -32001, Message: fmt.Sprintf("task %q not found", params.ID)})
		return
	}

	writeJSONRPCWithID(w, req.ID, task, nil)
}

func (h *TaskHandler) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		for id, task := range h.tasks {
			if task.Status.State == TaskStateCompleted || task.Status.State == TaskStateFailed || task.Status.State == TaskStateCanceled {
				if time.Since(task.UpdatedAt) > 5*time.Minute {
					delete(h.tasks, id)
				}
			}
		}
		h.mu.Unlock()
	}
}

func writeJSONRPC(w http.ResponseWriter, result interface{}, err *JSONRPCError) {
	writeJSONRPCWithID(w, nil, result, err)
}

func writeJSONRPCWithID(w http.ResponseWriter, id json.RawMessage, result interface{}, rpcErr *JSONRPCError) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
