// Package controlplane provides the Volra control plane API server and persistence.
package controlplane

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Agent represents a registered agent in the control plane.
type Agent struct {
	Name         string     `json:"name"`
	Dir          string     `json:"dir"`
	Framework    string     `json:"framework,omitempty"`
	Port         int        `json:"port,omitempty"`
	HealthPath   string     `json:"health_path,omitempty"`
	Status       string     `json:"status"`
	LastDeployAt *time.Time `json:"last_deploy_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// APIKey represents an API key for authentication.
type APIKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	Role      string     `json:"role"`
	Namespace string     `json:"namespace,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// FederationPeer represents a remote control plane.
type FederationPeer struct {
	URL     string    `json:"url"`
	Name    string    `json:"name"`
	APIKey  string    `json:"-"`
	AddedAt time.Time `json:"added_at"`
}

const schema = `
CREATE TABLE IF NOT EXISTS agents (
    name TEXT PRIMARY KEY,
    dir TEXT NOT NULL,
    framework TEXT DEFAULT '',
    port INTEGER DEFAULT 0,
    health_path TEXT DEFAULT '',
    status TEXT DEFAULT 'unknown',
    last_deploy_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'operator', 'viewer')),
    namespace TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME
);

CREATE TABLE IF NOT EXISTS federation_peers (
    url TEXT PRIMARY KEY,
    name TEXT DEFAULT '',
    api_key TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// Store provides persistence for the control plane.
type Store struct {
	db *sql.DB
}

// DefaultDBPath returns the default database path (~/.volra/controlplane.db).
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".volra", "controlplane.db")
}

// NewStore opens or creates a SQLite database at the given path.
func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent access.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// --- Agent operations ---

// UpsertAgent inserts or updates an agent.
func (s *Store) UpsertAgent(a Agent) error {
	_, err := s.db.Exec(`
		INSERT INTO agents (name, dir, framework, port, health_path, status, last_deploy_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			dir = excluded.dir,
			framework = excluded.framework,
			port = excluded.port,
			health_path = excluded.health_path,
			status = excluded.status,
			last_deploy_at = excluded.last_deploy_at
	`, a.Name, a.Dir, a.Framework, a.Port, a.HealthPath, a.Status, a.LastDeployAt, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("upserting agent %s: %w", a.Name, err)
	}
	return nil
}

// GetAgent returns a single agent by name.
func (s *Store) GetAgent(name string) (*Agent, error) {
	row := s.db.QueryRow(`SELECT name, dir, framework, port, health_path, status, last_deploy_at, created_at FROM agents WHERE name = ?`, name)
	a := &Agent{}
	if err := row.Scan(&a.Name, &a.Dir, &a.Framework, &a.Port, &a.HealthPath, &a.Status, &a.LastDeployAt, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying agent %s: %w", name, err)
	}
	return a, nil
}

// ListAgents returns all registered agents.
func (s *Store) ListAgents() ([]Agent, error) {
	rows, err := s.db.Query(`SELECT name, dir, framework, port, health_path, status, last_deploy_at, created_at FROM agents ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.Name, &a.Dir, &a.Framework, &a.Port, &a.HealthPath, &a.Status, &a.LastDeployAt, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning agent row: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// DeleteAgent removes an agent by name.
func (s *Store) DeleteAgent(name string) error {
	_, err := s.db.Exec(`DELETE FROM agents WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("deleting agent %s: %w", name, err)
	}
	return nil
}

// UpdateAgentStatus updates only the status field.
func (s *Store) UpdateAgentStatus(name, status string) error {
	_, err := s.db.Exec(`UPDATE agents SET status = ? WHERE name = ?`, status, name)
	if err != nil {
		return fmt.Errorf("updating agent status %s: %w", name, err)
	}
	return nil
}

// --- Import from legacy registry ---

// legacyRegistryEntry mirrors the existing ~/.volra/agents.json format.
type legacyRegistryEntry struct {
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	PrometheusPort int    `json:"prometheus_port"`
	AgentPort      int    `json:"agent_port"`
}

// ImportFromLegacyRegistry reads ~/.volra/agents.json and imports agents into SQLite.
func (s *Store) ImportFromLegacyRegistry(registryPath string) (int, error) {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading legacy registry: %w", err)
	}

	var entries []legacyRegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return 0, fmt.Errorf("parsing legacy registry: %w", err)
	}

	imported := 0
	for _, e := range entries {
		// Only import if not already in the database.
		existing, err := s.GetAgent(e.Name)
		if err != nil {
			return imported, err
		}
		if existing != nil {
			continue
		}

		a := Agent{
			Name:      e.Name,
			Dir:       e.Dir,
			Port:      e.AgentPort,
			Status:    "unknown",
			CreatedAt: time.Now().UTC(),
		}
		if err := s.UpsertAgent(a); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

// --- API Key operations ---

// InsertAPIKey stores a new API key.
func (s *Store) InsertAPIKey(key APIKey) error {
	_, err := s.db.Exec(`INSERT INTO api_keys (id, name, key_hash, role, namespace, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		key.ID, key.Name, key.KeyHash, key.Role, key.Namespace, key.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting API key: %w", err)
	}
	return nil
}

// ListAPIKeys returns all non-revoked API keys.
func (s *Store) ListAPIKeys() ([]APIKey, error) {
	rows, err := s.db.Query(`SELECT id, name, key_hash, role, namespace, created_at, revoked_at FROM api_keys ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.Role, &k.Namespace, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, fmt.Errorf("scanning API key row: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// GetAPIKeyByHash returns the API key matching the given hash, or nil.
func (s *Store) GetAPIKeyByID(id string) (*APIKey, error) {
	row := s.db.QueryRow(`SELECT id, name, key_hash, role, namespace, created_at, revoked_at FROM api_keys WHERE id = ? AND revoked_at IS NULL`, id)
	var k APIKey
	if err := row.Scan(&k.ID, &k.Name, &k.KeyHash, &k.Role, &k.Namespace, &k.CreatedAt, &k.RevokedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying API key: %w", err)
	}
	return &k, nil
}

// GetActiveAPIKeys returns all non-revoked API keys (for auth verification).
func (s *Store) GetActiveAPIKeys() ([]APIKey, error) {
	rows, err := s.db.Query(`SELECT id, name, key_hash, role, namespace, created_at FROM api_keys WHERE revoked_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("listing active API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.Role, &k.Namespace, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning API key row: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RevokeAPIKey marks an API key as revoked.
func (s *Store) RevokeAPIKey(id string) error {
	now := time.Now().UTC()
	result, err := s.db.Exec(`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("revoking API key: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("API key %s not found or already revoked", id)
	}
	return nil
}

// HasAPIKeys returns true if any API keys exist (for RBAC auto-enable).
func (s *Store) HasAPIKeys() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE revoked_at IS NULL`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting API keys: %w", err)
	}
	return count > 0, nil
}

// --- Federation operations ---

// InsertPeer adds a federation peer.
func (s *Store) InsertPeer(p FederationPeer) error {
	_, err := s.db.Exec(`INSERT INTO federation_peers (url, name, api_key, added_at) VALUES (?, ?, ?, ?)`,
		p.URL, p.Name, p.APIKey, p.AddedAt)
	if err != nil {
		return fmt.Errorf("inserting federation peer: %w", err)
	}
	return nil
}

// ListPeers returns all federation peers.
func (s *Store) ListPeers() ([]FederationPeer, error) {
	rows, err := s.db.Query(`SELECT url, name, api_key, added_at FROM federation_peers ORDER BY added_at`)
	if err != nil {
		return nil, fmt.Errorf("listing federation peers: %w", err)
	}
	defer rows.Close()

	var peers []FederationPeer
	for rows.Next() {
		var p FederationPeer
		if err := rows.Scan(&p.URL, &p.Name, &p.APIKey, &p.AddedAt); err != nil {
			return nil, fmt.Errorf("scanning federation peer row: %w", err)
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// DeletePeer removes a federation peer.
func (s *Store) DeletePeer(url string) error {
	result, err := s.db.Exec(`DELETE FROM federation_peers WHERE url = ?`, url)
	if err != nil {
		return fmt.Errorf("deleting federation peer: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("federation peer %s not found", url)
	}
	return nil
}
