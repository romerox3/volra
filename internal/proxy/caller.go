// Package proxy implements the A2A smart sidecar reverse proxy.
// It translates incoming A2A Tasks/send requests into HTTP calls to the agent.
package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
)

// AgentCaller translates A2A messages into HTTP requests to the agent.
type AgentCaller struct {
	agentURL string
	mode     agentfile.A2AMode
	skills   map[string]agentfile.A2ASkill
	client   *http.Client
}

// NewAgentCaller creates a caller for the given agent URL and A2A config.
func NewAgentCaller(agentURL string, mode agentfile.A2AMode, skills []agentfile.A2ASkill) *AgentCaller {
	if mode == "" {
		mode = agentfile.A2AModeDefault
	}
	skillMap := make(map[string]agentfile.A2ASkill, len(skills))
	for _, s := range skills {
		skillMap[s.ID] = s
	}
	return &AgentCaller{
		agentURL: strings.TrimRight(agentURL, "/"),
		mode:     mode,
		skills:   skillMap,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Call dispatches a request based on the configured mode.
// For default mode, skillID is ignored.
// For declarative mode, skillID selects the endpoint.
// For passthrough mode, rawBody is forwarded as-is.
func (c *AgentCaller) Call(skillID string, text string, rawBody []byte) (string, error) {
	switch c.mode {
	case agentfile.A2AModePassthrough:
		return c.callPassthrough(rawBody)
	case agentfile.A2AModeDeclarative:
		return c.callSkill(skillID, text)
	default:
		return c.callDefault(text)
	}
}

// callDefault sends POST /ask with {"question": "<text>"} and extracts "answer".
func (c *AgentCaller) callDefault(text string) (string, error) {
	body := map[string]string{"question": text}
	return c.doPost("/ask", body, "answer")
}

// callSkill looks up the skill and routes to its configured endpoint.
func (c *AgentCaller) callSkill(skillID string, text string) (string, error) {
	skill, ok := c.skills[skillID]
	if !ok {
		return "", fmt.Errorf("skill %q not found", skillID)
	}

	method := skill.Method
	if method == "" {
		method = "POST"
	}
	reqField := skill.RequestField
	if reqField == "" {
		reqField = "question"
	}
	respField := skill.ResponseField
	if respField == "" {
		respField = "answer"
	}

	body := map[string]string{reqField: text}
	return c.doPost(skill.Endpoint, body, respField)
}

// callPassthrough forwards raw JSON-RPC to the agent's /a2a endpoint.
func (c *AgentCaller) callPassthrough(rawBody []byte) (string, error) {
	resp, err := c.client.Post(c.agentURL+"/a2a", "application/json", strings.NewReader(string(rawBody)))
	if err != nil {
		return "", fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading agent response: %w", err)
	}
	return string(data), nil
}

// doPost sends a POST with a JSON body and extracts a field from the response.
func (c *AgentCaller) doPost(endpoint string, body map[string]string, responseField string) (string, error) {
	payload, _ := json.Marshal(body)
	resp, err := c.client.Post(c.agentURL+endpoint, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading agent response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// If not JSON, return raw body.
		return string(data), nil
	}

	val, ok := result[responseField]
	if !ok {
		return string(data), nil
	}

	switch v := val.(type) {
	case string:
		return v, nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}
