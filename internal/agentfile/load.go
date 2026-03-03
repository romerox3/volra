package agentfile

import (
	"fmt"
	"os"
)

// Load reads an Agentfile from disk, parses it, and validates it.
// This is the primary entry point for consumers.
func Load(path string) (*Agentfile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening Agentfile: %w", err)
	}
	defer func() { _ = f.Close() }()

	af, err := Parse(f)
	if err != nil {
		return nil, err
	}

	if err := Validate(af); err != nil {
		return nil, err
	}

	return af, nil
}
