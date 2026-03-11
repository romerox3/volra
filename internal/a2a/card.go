// Package a2a implements the Agent-to-Agent (A2A) protocol support.
// It generates and serves Agent Cards at /.well-known/agent.json per the A2A spec.
package a2a

import (
	"encoding/json"
	"fmt"

	"github.com/romerox3/volra/internal/agentfile"
)

// AgentCard represents an A2A Agent Card (/.well-known/agent-card.json).
// See: https://a2a-protocol.org/v0.3.0/specification/
type AgentCard struct {
	Name              string          `json:"name"`
	Description       string          `json:"description,omitempty"`
	URL               string          `json:"url"`
	Version           string          `json:"version,omitempty"`
	DocumentVersion   string          `json:"documentVersion,omitempty"`
	Capabilities      *Capabilities   `json:"capabilities,omitempty"`
	Skills            []Skill         `json:"skills,omitempty"`
	Provider          *Provider       `json:"provider,omitempty"`
	Authentication    *Authentication `json:"authentication,omitempty"`
	DefaultInputModes  []string       `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string       `json:"defaultOutputModes,omitempty"`
}

// Authentication describes the authentication requirements for the agent.
type Authentication struct {
	Schemes  []string `json:"schemes"`
	Required bool     `json:"required"`
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
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// Provider identifies who provides the agent.
type Provider struct {
	Organization string `json:"organization,omitempty"`
	URL          string `json:"url,omitempty"`
}

// GenerateCard creates an A2A Agent Card from an Agentfile.
// It auto-enriches the card with Agentfile data: framework, observability, security.
func GenerateCard(af *agentfile.Agentfile, agentURL string) *AgentCard {
	skills := []Skill{
		{
			ID:          af.Name + "-primary",
			Name:        af.Name,
			Description: fmt.Sprintf("AI agent deployed via Volra (framework: %s)", af.Framework),
			Tags:        []string{string(af.Framework), "volra"},
			InputModes:  []string{"text"},
			OutputModes: []string{"text"},
		},
	}

	// If observability level 2 is configured, expose metrics as a skill.
	if af.Observability != nil && af.Observability.Level >= 2 {
		metricsPort := af.Observability.MetricsPort
		if metricsPort == 0 {
			metricsPort = 9101
		}
		skills = append(skills, Skill{
			ID:          af.Name + "-metrics",
			Name:        "metrics",
			Description: fmt.Sprintf("Prometheus metrics endpoint (port %d)", metricsPort),
			Tags:        []string{"observability", "prometheus", "volra"},
			InputModes:  []string{"text"},
			OutputModes: []string{"text"},
		})
	}

	// Add declarative skills from Agentfile a2a section.
	if af.A2A != nil && af.A2A.Mode == agentfile.A2AModeDeclarative {
		for _, s := range af.A2A.Skills {
			skills = append(skills, Skill{
				ID:          af.Name + "-" + s.ID,
				Name:        s.Name,
				Description: s.Description,
				Tags:        []string{"a2a", "volra"},
				InputModes:  []string{"text"},
				OutputModes: []string{"text"},
			})
		}
	}

	card := &AgentCard{
		Name:               af.Name,
		Description:        fmt.Sprintf("AI agent (%s) self-hosted via Volra", af.Framework),
		URL:                agentURL,
		Version:            "1.0.0",
		DocumentVersion:    "0.3.0",
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Provider: &Provider{
			Organization: "self-hosted",
		},
		Capabilities: &Capabilities{
			Streaming: false,
		},
		Skills: skills,
	}

	// Auto-apply authentication if security context is configured.
	if af.Security != nil {
		card = WithAuthentication(card)
	}

	return card
}

// WithAuthentication returns a copy of the card with bearer authentication configured.
func WithAuthentication(card *AgentCard) *AgentCard {
	card.Authentication = &Authentication{
		Schemes:  []string{"bearer"},
		Required: true,
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
	return `# A2A Agent Card (v0.3)
location = /.well-known/agent-card.json {
    alias /etc/volra/agent-card.json;
    default_type application/json;
}`
}
