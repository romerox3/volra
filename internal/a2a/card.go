// Package a2a implements the Agent-to-Agent (A2A) protocol support.
// It generates and serves Agent Cards at /.well-known/agent.json per the A2A spec.
package a2a

import (
	"encoding/json"
	"fmt"

	"github.com/romerox3/volra/internal/agentfile"
)

// AgentCard represents an A2A Agent Card (/.well-known/agent.json).
// See: https://google.github.io/A2A/specification/
type AgentCard struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	URL          string        `json:"url"`
	Version      string        `json:"version,omitempty"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
	Skills       []Skill       `json:"skills,omitempty"`
	Provider     *Provider     `json:"provider,omitempty"`
}

// Capabilities describes what the agent supports.
type Capabilities struct {
	Streaming         bool `json:"streaming,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	StateTransition   bool `json:"stateTransitionHistory,omitempty"`
}

// Skill describes a specific capability of the agent.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Provider identifies who provides the agent.
type Provider struct {
	Organization string `json:"organization,omitempty"`
	URL          string `json:"url,omitempty"`
}

// GenerateCard creates an A2A Agent Card from an Agentfile.
func GenerateCard(af *agentfile.Agentfile, agentURL string) *AgentCard {
	card := &AgentCard{
		Name:    af.Name,
		URL:     agentURL,
		Version: "1.0.0",
		Capabilities: &Capabilities{
			Streaming: false,
		},
		Skills: []Skill{
			{
				ID:          af.Name + "-primary",
				Name:        af.Name,
				Description: fmt.Sprintf("AI agent deployed via Volra (framework: %s)", af.Framework),
				Tags:        []string{string(af.Framework), "volra"},
			},
		},
	}
	return card
}

// RenderJSON serializes the Agent Card to pretty-printed JSON.
func RenderJSON(card *AgentCard) (string, error) {
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding agent card: %w", err)
	}
	return string(data), nil
}

// NginxLocationBlock returns an Nginx config snippet that serves the agent card
// at /.well-known/agent.json.
func NginxLocationBlock() string {
	return `# A2A Agent Card
location = /.well-known/agent.json {
    alias /etc/volra/agent-card.json;
    default_type application/json;
}`
}
