package a2a

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCaller struct {
	result string
	err    error
}

func (m *mockCaller) CallTool(_ string, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

func sendA2A(t *testing.T, handler http.Handler, method string, params interface{}) *httptest.ResponseRecorder {
	t.Helper()
	paramsJSON, _ := json.Marshal(params)
	body := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  method,
		Params:  paramsJSON,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestTaskHandler_TasksSend(t *testing.T) {
	handler := NewTaskHandler(&mockCaller{result: "Hello from agent"})

	params := TaskSendParams{
		Message: Message{
			Role:  "user",
			Parts: []Part{{Type: "text", Text: "do something"}},
		},
	}
	w := sendA2A(t, handler, "Tasks/send", params)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	// Parse result as Task.
	resultJSON, _ := json.Marshal(resp.Result)
	var task Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))

	assert.Equal(t, TaskStateCompleted, task.Status.State)
	require.Len(t, task.Artifacts, 1)
	assert.Equal(t, "Hello from agent", task.Artifacts[0].Parts[0].Text)
}

func TestTaskHandler_TasksGet(t *testing.T) {
	handler := NewTaskHandler(&mockCaller{result: "ok"})

	// First create a task.
	params := TaskSendParams{
		Message: Message{
			Role:  "user",
			Parts: []Part{{Type: "text", Text: "test"}},
		},
	}
	w := sendA2A(t, handler, "Tasks/send", params)

	var sendResp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sendResp))
	resultJSON, _ := json.Marshal(sendResp.Result)
	var task Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))

	// Now get the task.
	getParams := TaskGetParams{ID: task.ID}
	w2 := sendA2A(t, handler, "Tasks/get", getParams)

	var getResp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &getResp))
	assert.Nil(t, getResp.Error)
}

func TestTaskHandler_TasksGet_NotFound(t *testing.T) {
	handler := NewTaskHandler(nil)

	w := sendA2A(t, handler, "Tasks/get", TaskGetParams{ID: "nonexistent"})

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32001, resp.Error.Code)
}

func TestTaskHandler_TasksCancel(t *testing.T) {
	// We need a task in "working" state. Since our handler processes synchronously,
	// we test cancel on a completed task (won't change state) and verify no error.
	handler := NewTaskHandler(&mockCaller{result: "ok"})

	params := TaskSendParams{
		Message: Message{Role: "user", Parts: []Part{{Type: "text", Text: "test"}}},
	}
	w := sendA2A(t, handler, "Tasks/send", params)

	var sendResp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sendResp))
	resultJSON, _ := json.Marshal(sendResp.Result)
	var task Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))

	// Cancel the completed task (should return it unchanged).
	w2 := sendA2A(t, handler, "Tasks/cancel", TaskGetParams{ID: task.ID})
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestTaskHandler_UnknownMethod(t *testing.T) {
	handler := NewTaskHandler(nil)

	w := sendA2A(t, handler, "unknown/method", nil)

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestTaskHandler_InvalidJSON(t *testing.T) {
	handler := NewTaskHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32700, resp.Error.Code)
}

func TestTaskHandler_MethodNotAllowed(t *testing.T) {
	handler := NewTaskHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/a2a", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32600, resp.Error.Code)
}

func TestTaskHandler_CallerError(t *testing.T) {
	handler := NewTaskHandler(&mockCaller{err: assert.AnError})

	params := TaskSendParams{
		Message: Message{Role: "user", Parts: []Part{{Type: "text", Text: "fail"}}},
	}
	w := sendA2A(t, handler, "Tasks/send", params)

	var resp JSONRPCResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)

	resultJSON, _ := json.Marshal(resp.Result)
	var task Task
	require.NoError(t, json.Unmarshal(resultJSON, &task))
	assert.Equal(t, TaskStateFailed, task.Status.State)
}
