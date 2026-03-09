package dev

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeAgentfile(t *testing.T, dir string) {
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

// mockChecker returns a ComposeVersionChecker that returns a fixed version.
func mockChecker(version string, err error) ComposeVersionChecker {
	return func(_ context.Context, _ docker.DockerRunner) (string, error) {
		return version, err
	}
}

// noopExecutor records the call but does nothing.
func noopExecutor(composePath, projectName *string) WatchExecutor {
	return func(_ context.Context, cp, pn string) error {
		if composePath != nil {
			*composePath = cp
		}
		if projectName != nil {
			*projectName = pn
		}
		return nil
	}
}

func TestRun_NoAgentfile(t *testing.T) {
	dir := t.TempDir()
	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("2.32.4", nil), noopExecutor(nil, nil))
	require.Error(t, err)
}

func TestRun_ComposeVersionCheckFails(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir)
	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("", assert.AnError), noopExecutor(nil, nil))
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeComposeWatchRequired, ue.Code)
}

func TestRun_ComposeTooOld(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir)
	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("2.20.3", nil), noopExecutor(nil, nil))
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeComposeWatchRequired, ue.Code)
	assert.Contains(t, ue.What, "2.20.3")
}

func TestRun_GeneratesArtifacts(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')"), 0644))

	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("2.32.4", nil), noopExecutor(nil, nil))
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(dir, ".volra", "docker-compose.yml"))
	assert.FileExists(t, filepath.Join(dir, ".volra", "Dockerfile"))
	assert.FileExists(t, filepath.Join(dir, ".volra", "prometheus.yml"))
}

func TestRun_ComposeYAMLContainsWatchSection(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')"), 0644))

	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("2.32.4", nil), noopExecutor(nil, nil))
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, ".volra", "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "develop:")
	assert.Contains(t, string(content), "watch:")
	assert.Contains(t, string(content), "action: rebuild")
	assert.Contains(t, string(content), "__pycache__/")
}

func TestRun_PassesCorrectArgsToExecutor(t *testing.T) {
	dir := t.TempDir()
	writeAgentfile(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')"), 0644))

	var gotComposePath, gotProjectName string
	p := output.NewPresenter(output.ModePlain)
	dr := &testutil.MockDockerRunner{Responses: make(map[string]testutil.MockResponse)}

	err := Run(context.Background(), dir, p, dr, mockChecker("2.32.4", nil), noopExecutor(&gotComposePath, &gotProjectName))
	require.NoError(t, err)

	assert.Contains(t, gotComposePath, ".volra/docker-compose.yml")
	assert.Equal(t, "test-agent", gotProjectName)
}

func TestIsComposeWatchSupported(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.32.4", true},
		{"2.22.0", true},
		{"2.22.1", true},
		{"2.23.0", true},
		{"3.0.0", true},
		{"2.21.9", false},
		{"2.20.3", false},
		{"1.29.0", false},
		{"v2.29.1", true},
		{"2.22.0-beta.1", true},
		{"  2.32.4\n", true},
		{"garbage", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsComposeWatchSupported(tt.version))
		})
	}
}
