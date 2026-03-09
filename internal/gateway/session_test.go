package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_CreateNew(t *testing.T) {
	sm := NewSessionManager()

	s, isNew := sm.GetOrCreate("")
	assert.True(t, isNew)
	require.NotEmpty(t, s.ID)
	assert.Equal(t, 1, sm.Count())
}

func TestSessionManager_ReuseExisting(t *testing.T) {
	sm := NewSessionManager()

	s1, _ := sm.GetOrCreate("")
	s2, isNew := sm.GetOrCreate(s1.ID)
	assert.False(t, isNew)
	assert.Equal(t, s1.ID, s2.ID)
	assert.Equal(t, 1, sm.Count())
}

func TestSessionManager_UnknownIDCreatesNew(t *testing.T) {
	sm := NewSessionManager()

	s, isNew := sm.GetOrCreate("nonexistent-id")
	assert.True(t, isNew)
	assert.NotEqual(t, "nonexistent-id", s.ID)
	assert.Equal(t, 1, sm.Count())
}

func TestSessionManager_RecordInteraction(t *testing.T) {
	sm := NewSessionManager()

	s, _ := sm.GetOrCreate("")
	sm.RecordInteraction(s.ID, "agent-a")
	sm.RecordInteraction(s.ID, "agent-b")

	assert.True(t, s.AgentBackends["agent-a"])
	assert.True(t, s.AgentBackends["agent-b"])
}

func TestSessionManager_Remove(t *testing.T) {
	sm := NewSessionManager()

	s, _ := sm.GetOrCreate("")
	assert.Equal(t, 1, sm.Count())

	sm.Remove(s.ID)
	assert.Equal(t, 0, sm.Count())
}

func TestSessionManager_RecordInteractionIgnoresUnknown(t *testing.T) {
	sm := NewSessionManager()
	// Should not panic.
	sm.RecordInteraction("nonexistent", "agent-a")
}
