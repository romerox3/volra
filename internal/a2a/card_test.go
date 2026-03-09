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

func TestNginxLocationBlock(t *testing.T) {
	block := NginxLocationBlock()
	assert.Contains(t, block, "/.well-known/agent.json")
	assert.Contains(t, block, "application/json")
}
