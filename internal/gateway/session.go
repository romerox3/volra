package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// SessionManager tracks MCP session-to-agent affinities.
// When a client establishes a session via Mcp-Session-Id, calls from that session
// maintain affinity to the backends they've interacted with.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// Session holds state for a single MCP client session.
type Session struct {
	ID string `json:"id"`
	// AgentBackends maps agent names to any session-specific state needed.
	// For now we just track which agents this session has talked to.
	AgentBackends map[string]bool `json:"agent_backends"`
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate returns an existing session or creates a new one.
// If sessionID is empty, generates a new one.
func (sm *SessionManager) GetOrCreate(sessionID string) (*Session, bool) {
	if sessionID == "" {
		return sm.create(), true
	}

	sm.mu.RLock()
	s, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()
	if ok {
		return s, false
	}

	return sm.create(), true
}

// RecordInteraction marks that a session has interacted with an agent.
func (sm *SessionManager) RecordInteraction(sessionID, agentName string) {
	sm.mu.RLock()
	s, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()
	if !ok {
		return
	}

	sm.mu.Lock()
	s.AgentBackends[agentName] = true
	sm.mu.Unlock()
}

// Remove deletes a session.
func (sm *SessionManager) Remove(sessionID string) {
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

func (sm *SessionManager) create() *Session {
	s := &Session{
		ID:            generateSessionID(),
		AgentBackends: make(map[string]bool),
	}
	sm.mu.Lock()
	sm.sessions[s.ID] = s
	sm.mu.Unlock()
	return s
}

// generateSessionID creates a random 16-byte hex session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
