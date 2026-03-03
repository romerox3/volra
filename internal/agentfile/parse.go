package agentfile

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Parse reads YAML from the given reader and returns a syntactically valid Agentfile.
// It uses strict mode (KnownFields) to reject unknown fields.
func Parse(r io.Reader) (*Agentfile, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	var af Agentfile
	if err := dec.Decode(&af); err != nil {
		return nil, fmt.Errorf("parsing Agentfile: %w", err)
	}

	return &af, nil
}
