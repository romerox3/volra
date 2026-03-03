package testutil

import (
	"context"
	"fmt"
	"strings"
)

// MockResponse holds a preconfigured response for MockDockerRunner.
type MockResponse struct {
	Output string
	Err    error
}

// MockDockerRunner implements docker.DockerRunner with configurable responses.
type MockDockerRunner struct {
	Responses map[string]MockResponse
	Calls     [][]string
}

// Run looks up the space-joined args in Responses and returns the configured output.
// All calls are recorded in Calls for assertion.
func (m *MockDockerRunner) Run(_ context.Context, args ...string) (string, error) {
	m.Calls = append(m.Calls, args)
	key := strings.Join(args, " ")
	if r, ok := m.Responses[key]; ok {
		return r.Output, r.Err
	}
	return "", fmt.Errorf("unexpected docker call: %s", key)
}
