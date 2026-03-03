package testutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockDockerRunner_ReturnsConfiguredResponse(t *testing.T) {
	m := &MockDockerRunner{
		Responses: map[string]MockResponse{
			"info":            {Output: "Client: Docker Engine", Err: nil},
			"compose version": {Output: "Docker Compose version v2.32.4", Err: nil},
		},
	}

	out, err := m.Run(context.Background(), "info")
	require.NoError(t, err)
	assert.Equal(t, "Client: Docker Engine", out)

	out, err = m.Run(context.Background(), "compose", "version")
	require.NoError(t, err)
	assert.Equal(t, "Docker Compose version v2.32.4", out)
}

func TestMockDockerRunner_ReturnsError(t *testing.T) {
	m := &MockDockerRunner{
		Responses: map[string]MockResponse{
			"info": {Output: "", Err: errors.New("docker not running")},
		},
	}

	out, err := m.Run(context.Background(), "info")
	assert.Error(t, err)
	assert.Equal(t, "", out)
	assert.Contains(t, err.Error(), "docker not running")
}

func TestMockDockerRunner_ErrorsOnUnexpectedCall(t *testing.T) {
	m := &MockDockerRunner{
		Responses: map[string]MockResponse{},
	}

	_, err := m.Run(context.Background(), "unknown", "command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected docker call: unknown command")
}

func TestMockDockerRunner_RecordsCalls(t *testing.T) {
	m := &MockDockerRunner{
		Responses: map[string]MockResponse{
			"info":            {Output: "", Err: nil},
			"compose version": {Output: "", Err: nil},
		},
	}

	_, _ = m.Run(context.Background(), "info")
	_, _ = m.Run(context.Background(), "compose", "version")

	require.Len(t, m.Calls, 2)
	assert.Equal(t, []string{"info"}, m.Calls[0])
	assert.Equal(t, []string{"compose", "version"}, m.Calls[1])
}
