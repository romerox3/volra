// Package audit provides an append-only audit log for Volra operations.
// Entries are stored as JSON Lines in .volra/audit.log.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single audit log entry.
type Entry struct {
	Timestamp  time.Time      `json:"timestamp"`
	Action     string         `json:"action"`
	Agent      string         `json:"agent"`
	User       string         `json:"user"`
	Result     string         `json:"result"`
	DurationMs int64          `json:"duration_ms"`
	Details    map[string]any `json:"details,omitempty"`
}

const auditFile = ".volra/audit.log"

// Append writes a new entry to the audit log. Creates the file if it doesn't exist.
func Append(dir string, entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if entry.User == "" {
		entry.User = os.Getenv("USER")
	}

	path := filepath.Join(dir, auditFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating audit directory: %w", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding audit entry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing audit entry: %w", err)
	}
	return nil
}

// Filter defines criteria for filtering audit entries.
type Filter struct {
	Action string
	Agent  string
	Since  time.Time
}

// Read returns all audit entries from the log, optionally filtered.
func Read(dir string, filter *Filter) ([]Entry, error) {
	path := filepath.Join(dir, auditFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading audit log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var entries []Entry

	for _, line := range lines {
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		if filter != nil {
			if filter.Action != "" && e.Action != filter.Action {
				continue
			}
			if filter.Agent != "" && e.Agent != filter.Agent {
				continue
			}
			if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
				continue
			}
		}
		entries = append(entries, e)
	}

	return entries, nil
}
