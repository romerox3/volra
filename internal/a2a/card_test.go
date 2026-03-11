package a2a

import (
	"encoding/json"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCard(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "my-agent",
		Framework: agentfile.FrameworkLangGraph,
		Port:      8000,
	}

	card := GenerateCard(af, "http://localhost:8000")

	assert.Equal(t, "my-agent", card.Name)
	assert.Equal(t, "http://localhost:8000", card.URL)
	assert.Equal(t, "1.0.0", card.Version)
	require.NotNil(t, card.Capabilities)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, "my-agent-primary", card.Skills[0].ID)
	assert.Contains(t, card.Skills[0].Description, "langgraph")
	assert.Contains(t, card.Skills[0].Tags, "langgraph")
	assert.Contains(t, card.Skills[0].Tags, "volra")
}

func TestGenerateCard_V03Fields(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "test-agent",
		Framework: agentfile.FrameworkGeneric,
	}

	card := GenerateCard(af, "http://localhost:8000")

	// DocumentVersion
	assert.Equal(t, "0.3.0", card.DocumentVersion)

	// Default modes
	assert.Equal(t, []string{"text"}, card.DefaultInputModes)
	assert.Equal(t, []string{"text"}, card.DefaultOutputModes)

	// Skill modes
	require.Len(t, card.Skills, 1)
	assert.Equal(t, []string{"text"}, card.Skills[0].InputModes)
	assert.Equal(t, []string{"text"}, card.Skills[0].OutputModes)

	// No authentication by default
	assert.Nil(t, card.Authentication)
}

func TestWithAuthentication(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "secure-agent",
		Framework: agentfile.FrameworkGeneric,
	}

	card := GenerateCard(af, "http://localhost:8000")
	card = WithAuthentication(card)

	require.NotNil(t, card.Authentication)
	assert.Equal(t, []string{"bearer"}, card.Authentication.Schemes)
	assert.True(t, card.Authentication.Required)
}

func TestRenderJSON(t *testing.T) {
	card := &AgentCard{
		Name: "test-agent",
		URL:  "http://localhost:8000",
	}

	result, err := RenderJSON(card)
	require.NoError(t, err)

	// Verify it's valid JSON.
	var parsed AgentCard
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "test-agent", parsed.Name)
}

func TestRenderJSON_V03Roundtrip(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "roundtrip-agent",
		Framework: agentfile.FrameworkLangGraph,
	}

	card := GenerateCard(af, "http://localhost:8000")
	card = WithAuthentication(card)

	jsonStr, err := RenderJSON(card)
	require.NoError(t, err)

	var parsed AgentCard
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

	assert.Equal(t, "0.3.0", parsed.DocumentVersion)
	assert.Equal(t, []string{"text"}, parsed.DefaultInputModes)
	assert.Equal(t, []string{"text"}, parsed.DefaultOutputModes)
	require.NotNil(t, parsed.Authentication)
	assert.Equal(t, []string{"bearer"}, parsed.Authentication.Schemes)
	assert.True(t, parsed.Authentication.Required)
	require.Len(t, parsed.Skills, 1)
	assert.Equal(t, []string{"text"}, parsed.Skills[0].InputModes)
}

func TestRenderJSON_OmitsEmptyAuth(t *testing.T) {
	card := &AgentCard{
		Name: "no-auth-agent",
		URL:  "http://localhost:8000",
	}

	result, err := RenderJSON(card)
	require.NoError(t, err)
	assert.NotContains(t, result, "authentication")
}

func TestGenerateCard_GenericFramework(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "generic-bot",
		Framework: agentfile.FrameworkGeneric,
	}

	card := GenerateCard(af, "http://localhost:9000")
	assert.Contains(t, card.Skills[0].Tags, "generic")
}

func TestGenerateCard_CrewAI(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "crew-bot",
		Framework: agentfile.FrameworkCrewAI,
	}

	card := GenerateCard(af, "http://localhost:9000")
	assert.Contains(t, card.Skills[0].Tags, "crewai")
}

func TestGenerateCard_WithObservabilityLevel2(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "obs-agent",
		Framework: agentfile.FrameworkGeneric,
		Observability: &agentfile.ObservabilityConfig{
			Level:       2,
			MetricsPort: 9101,
		},
	}

	card := GenerateCard(af, "http://localhost:8000")

	require.Len(t, card.Skills, 2)
	assert.Equal(t, "obs-agent-primary", card.Skills[0].ID)
	assert.Equal(t, "obs-agent-metrics", card.Skills[1].ID)
	assert.Contains(t, card.Skills[1].Tags, "prometheus")
}

func TestGenerateCard_WithObservabilityLevel2_DefaultPort(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "obs-agent",
		Framework: agentfile.FrameworkGeneric,
		Observability: &agentfile.ObservabilityConfig{
			Level: 2,
		},
	}

	card := GenerateCard(af, "http://localhost:8000")
	require.Len(t, card.Skills, 2)
	assert.Contains(t, card.Skills[1].Description, "9101")
}

func TestGenerateCard_WithSecurityAutoAuth(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "secure-agent",
		Framework: agentfile.FrameworkGeneric,
		Security: &agentfile.SecurityContext{
			ReadOnly: true,
		},
	}

	card := GenerateCard(af, "http://localhost:8000")

	require.NotNil(t, card.Authentication)
	assert.Equal(t, []string{"bearer"}, card.Authentication.Schemes)
	assert.True(t, card.Authentication.Required)
}

func TestGenerateCard_Provider(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "my-agent",
		Framework: agentfile.FrameworkGeneric,
	}

	card := GenerateCard(af, "http://localhost:8000")

	require.NotNil(t, card.Provider)
	assert.Equal(t, "self-hosted", card.Provider.Organization)
}

func TestGenerateCard_Description(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "my-agent",
		Framework: agentfile.FrameworkLangGraph,
	}

	card := GenerateCard(af, "http://localhost:8000")
	assert.Contains(t, card.Description, "langgraph")
	assert.Contains(t, card.Description, "self-hosted")
}

func TestGenerateCard_WithA2ASkills(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "my-agent",
		Framework: agentfile.FrameworkGeneric,
		A2A: &agentfile.A2AConfig{
			Mode: agentfile.A2AModeDeclarative,
			Skills: []agentfile.A2ASkill{
				{ID: "research", Name: "research", Description: "Research capability"},
				{ID: "search", Name: "semantic-search", Description: "Search docs"},
			},
		},
	}

	card := GenerateCard(af, "http://localhost:8000")

	// 1 primary + 2 declarative = 3
	require.Len(t, card.Skills, 3)
	assert.Equal(t, "my-agent-primary", card.Skills[0].ID)
	assert.Equal(t, "my-agent-research", card.Skills[1].ID)
	assert.Equal(t, "Research capability", card.Skills[1].Description)
	assert.Contains(t, card.Skills[1].Tags, "a2a")
	assert.Equal(t, "my-agent-search", card.Skills[2].ID)
}

func TestGenerateCard_WithA2ASkills_NotDeclarative(t *testing.T) {
	af := &agentfile.Agentfile{
		Name:      "my-agent",
		Framework: agentfile.FrameworkGeneric,
		A2A: &agentfile.A2AConfig{
			Mode: agentfile.A2AModeDefault,
		},
	}

	card := GenerateCard(af, "http://localhost:8000")
	// Only primary skill, no declarative skills added.
	require.Len(t, card.Skills, 1)
}

func TestNginxLocationBlock(t *testing.T) {
	block := NginxLocationBlock()
	assert.Contains(t, block, "/.well-known/agent-card.json")
	assert.Contains(t, block, "application/json")
}
