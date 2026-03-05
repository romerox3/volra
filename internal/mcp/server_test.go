package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sendRequest encodes a JSON-RPC request as a single line.
func sendRequest(t *testing.T, method string, id any, params any) string {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id != nil {
		req["id"] = id
	}
	if params != nil {
		req["params"] = params
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	return string(b)
}

// runMCP sends one or more request lines and returns all response lines.
func runMCP(t *testing.T, lines ...string) []Response {
	t.Helper()
	input := strings.Join(lines, "\n") + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := Serve(context.TODO(), in, &out, "0.0.0-test")
	require.NoError(t, err)

	var responses []Response
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var resp Response
		require.NoError(t, json.Unmarshal([]byte(line), &resp), "failed to parse: %s", line)
		responses = append(responses, resp)
	}
	return responses
}

func TestServe_Initialize(t *testing.T) {
	req := sendRequest(t, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo":      map[string]any{"name": "test-editor", "version": "1.0"},
	})

	responses := runMCP(t, req)
	require.Len(t, responses, 1)

	resp := responses[0]
	assert.Nil(t, resp.Error)

	// Parse result as InitializeResult
	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result InitializeResult
	require.NoError(t, json.Unmarshal(b, &result))

	assert.Equal(t, MCPProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, "volra", result.ServerInfo.Name)
	assert.Equal(t, "0.0.0-test", result.ServerInfo.Version)
	assert.NotNil(t, result.Capabilities.Tools)
}

func TestServe_ToolsList(t *testing.T) {
	req := sendRequest(t, "tools/list", 2, nil)
	responses := runMCP(t, req)
	require.Len(t, responses, 1)

	resp := responses[0]
	assert.Nil(t, resp.Error)

	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result ToolsListResult
	require.NoError(t, json.Unmarshal(b, &result))

	// Should have 4 tools
	assert.Len(t, result.Tools, 4)

	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	assert.Contains(t, names, "volra_deploy")
	assert.Contains(t, names, "volra_status")
	assert.Contains(t, names, "volra_logs")
	assert.Contains(t, names, "volra_doctor")
}

func TestServe_ToolsList_HasDescriptions(t *testing.T) {
	req := sendRequest(t, "tools/list", 3, nil)
	responses := runMCP(t, req)
	require.Len(t, responses, 1)

	b, _ := json.Marshal(responses[0].Result)
	var result ToolsListResult
	_ = json.Unmarshal(b, &result)

	for _, tool := range result.Tools {
		assert.NotEmpty(t, tool.Description, "tool %s should have a description", tool.Name)
		assert.NotNil(t, tool.InputSchema, "tool %s should have an inputSchema", tool.Name)
	}
}

func TestServe_NotificationIgnored(t *testing.T) {
	// notifications/initialized should not produce a response
	init := sendRequest(t, "initialize", 1, nil)
	notif := sendRequest(t, "notifications/initialized", nil, nil)
	list := sendRequest(t, "tools/list", 2, nil)

	responses := runMCP(t, init, notif, list)
	// Should only get 2 responses (initialize + tools/list), not 3
	assert.Len(t, responses, 2)
}

func TestServe_UnknownMethod(t *testing.T) {
	req := sendRequest(t, "nonexistent/method", 99, nil)
	responses := runMCP(t, req)
	require.Len(t, responses, 1)

	resp := responses[0]
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "nonexistent/method")
}

func TestServe_ParseError(t *testing.T) {
	in := strings.NewReader("this is not json\n")
	var out bytes.Buffer

	err := Serve(context.TODO(), in, &out, "0.0.0-test")
	require.NoError(t, err) // server should not crash

	var resp Response
	require.NoError(t, json.Unmarshal(out.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeParseError, resp.Error.Code)
}

func TestServe_EmptyLines(t *testing.T) {
	// Empty lines should be skipped
	req := sendRequest(t, "tools/list", 1, nil)
	input := "\n\n" + req + "\n\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := Serve(context.TODO(), in, &out, "0.0.0-test")
	require.NoError(t, err)

	var resp Response
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp))
	assert.Nil(t, resp.Error)
}

func TestServe_MultipleRequests(t *testing.T) {
	init := sendRequest(t, "initialize", 1, nil)
	list := sendRequest(t, "tools/list", 2, nil)

	responses := runMCP(t, init, list)
	assert.Len(t, responses, 2)

	// First should be initialize response
	b1, _ := json.Marshal(responses[0].Result)
	var initResult InitializeResult
	_ = json.Unmarshal(b1, &initResult)
	assert.Equal(t, "volra", initResult.ServerInfo.Name)

	// Second should be tools list
	b2, _ := json.Marshal(responses[1].Result)
	var toolsResult ToolsListResult
	_ = json.Unmarshal(b2, &toolsResult)
	assert.Len(t, toolsResult.Tools, 4)
}

func TestServe_ToolsCall_UnknownTool(t *testing.T) {
	req := sendRequest(t, "tools/call", 5, map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{},
	})

	responses := runMCP(t, req)
	require.Len(t, responses, 1)

	// Should return a tool result with isError=true, not a JSON-RPC error
	resp := responses[0]
	assert.Nil(t, resp.Error)

	b, _ := json.Marshal(resp.Result)
	var result ToolCallResult
	require.NoError(t, json.Unmarshal(b, &result))
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Unknown tool")
}

func TestServe_ToolsCall_InvalidParams(t *testing.T) {
	// tools/call with malformed params
	raw := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":"not an object"}`
	in := strings.NewReader(raw + "\n")
	var out bytes.Buffer

	err := Serve(context.TODO(), in, &out, "0.0.0-test")
	require.NoError(t, err)

	var resp Response
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInvalidParams, resp.Error.Code)
}

func TestProtocol_TextContent(t *testing.T) {
	c := TextContent("hello")
	assert.Equal(t, "text", c.Type)
	assert.Equal(t, "hello", c.Text)
}

func TestProtocol_SuccessResult(t *testing.T) {
	r := SuccessResult("all good")
	assert.False(t, r.IsError)
	assert.Len(t, r.Content, 1)
	assert.Equal(t, "all good", r.Content[0].Text)
}

func TestProtocol_ErrorResult(t *testing.T) {
	r := ErrorResult("something broke")
	assert.True(t, r.IsError)
	assert.Len(t, r.Content, 1)
	assert.Equal(t, "something broke", r.Content[0].Text)
}
