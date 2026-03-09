// Package auth provides API key authentication and role-based access control.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role represents a user role for RBAC.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// ValidRoles is the set of allowed roles.
var ValidRoles = map[Role]bool{
	RoleAdmin:    true,
	RoleOperator: true,
	RoleViewer:   true,
}

// KeyResult is returned after creating a new API key.
type KeyResult struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // Only shown once at creation
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateKey creates a new API key with a random 32-byte hex value.
func GenerateKey() (id string, plaintext string, hash string, err error) {
	// Generate key ID (8 bytes = 16 hex chars).
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", "", fmt.Errorf("generating key ID: %w", err)
	}
	id = hex.EncodeToString(idBytes)

	// Generate key value (32 bytes = 64 hex chars).
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", "", fmt.Errorf("generating key value: %w", err)
	}
	plaintext = hex.EncodeToString(keyBytes)

	// Hash the key for storage.
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", fmt.Errorf("hashing key: %w", err)
	}
	hash = string(hashed)

	return id, plaintext, hash, nil
}

// VerifyKey checks if a plaintext key matches a bcrypt hash.
func VerifyKey(plaintext, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// CanPerform checks if a role has permission for an action.
func CanPerform(role Role, action Action) bool {
	perms, ok := permissions[role]
	if !ok {
		return false
	}
	return perms[action]
}

// Action represents an API action that requires authorization.
type Action string

const (
	ActionListAgents  Action = "list_agents"
	ActionGetAgent    Action = "get_agent"
	ActionDeployAgent Action = "deploy_agent"
	ActionStopAgent   Action = "stop_agent"
	ActionViewLogs    Action = "view_logs"
	ActionManageKeys  Action = "manage_keys"
	ActionManagePeers Action = "manage_peers"
)

// permissions defines what each role can do.
var permissions = map[Role]map[Action]bool{
	RoleAdmin: {
		ActionListAgents:  true,
		ActionGetAgent:    true,
		ActionDeployAgent: true,
		ActionStopAgent:   true,
		ActionViewLogs:    true,
		ActionManageKeys:  true,
		ActionManagePeers: true,
	},
	RoleOperator: {
		ActionListAgents:  true,
		ActionGetAgent:    true,
		ActionDeployAgent: true,
		ActionStopAgent:   true,
		ActionViewLogs:    true,
		ActionManageKeys:  false,
		ActionManagePeers: false,
	},
	RoleViewer: {
		ActionListAgents:  true,
		ActionGetAgent:    true,
		ActionDeployAgent: false,
		ActionStopAgent:   false,
		ActionViewLogs:    true,
		ActionManageKeys:  false,
		ActionManagePeers: false,
	},
}
