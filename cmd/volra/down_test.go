package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestAgentfile(t *testing.T, dir string) {
	t.Helper()
	content := `version: 1
name: test-agent
framework: generic
port: 8000
health_path: /health
dockerfile: auto
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Agentfile"), []byte(content), 0644))
}

func writeTestCompose(t *testing.T, dir string) {
	t.Helper()
	volraDir := filepath.Join(dir, deploy.OutputDir)
	require.NoError(t, os.MkdirAll(volraDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(volraDir, "docker-compose.yml"), []byte("version: '3'\n"), 0644))
}

func TestDown_NoComposeFile(t *testing.T) {
	dir := t.TempDir()
	writeTestAgentfile(t, dir)

	mock := &testutil.MockDockerRunner{}
	p := output.NewPresenter(output.ModePlain)

	err := runDown(context.Background(), dir, p, mock)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeNoDeploymentFound, ue.Code)
}

func mockForDown() *testutil.MockDockerRunner {
	return &testutil.MockDockerRunner{
		Responses: make(map[string]testutil.MockResponse),
	}
}

func TestDown_Success(t *testing.T) {
	dir := t.TempDir()
	writeTestAgentfile(t, dir)
	writeTestCompose(t, dir)

	mock := mockForDown()
	p := output.NewPresenter(output.ModePlain)

	// The mock returns error for unrecognized calls, so we need a wildcard approach.
	// Since the compose path is dynamic, we just check the error doesn't happen
	// by making the mock always succeed for any call.
	// Override Run to always succeed:
	err := runDown(context.Background(), dir, p, &successDockerRunner{calls: &mock.Calls})
	require.NoError(t, err)

	require.Len(t, mock.Calls, 1)
	args := mock.Calls[0]
	assert.Equal(t, "compose", args[0])
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "test-agent")
	assert.Contains(t, args, "down")
	assert.NotContains(t, args, "-v")
}

func TestDown_WithVolumes(t *testing.T) {
	dir := t.TempDir()
	writeTestAgentfile(t, dir)
	writeTestCompose(t, dir)

	var calls [][]string
	p := output.NewPresenter(output.ModePlain)

	removeVolumes = true
	defer func() { removeVolumes = false }()

	err := runDown(context.Background(), dir, p, &successDockerRunner{calls: &calls})
	require.NoError(t, err)

	require.Len(t, calls, 1)
	args := calls[0]
	assert.Contains(t, args, "-v")
}

func TestDown_DockerError(t *testing.T) {
	dir := t.TempDir()
	writeTestAgentfile(t, dir)
	writeTestCompose(t, dir)

	p := output.NewPresenter(output.ModePlain)

	err := runDown(context.Background(), dir, p, &errorDockerRunner{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopping services")
}

// successDockerRunner always succeeds and records calls.
type successDockerRunner struct {
	calls *[][]string
}

func (s *successDockerRunner) Run(_ context.Context, args ...string) (string, error) {
	*s.calls = append(*s.calls, args)
	return "", nil
}

// errorDockerRunner always returns an error.
type errorDockerRunner struct{}

func (e *errorDockerRunner) Run(_ context.Context, args ...string) (string, error) {
	return "", assert.AnError
}
