package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeDeployTestAgentfile(t *testing.T, dir string) {
	t.Helper()
	content := `version: 1
name: test-agent
framework: generic
port: 8000
health_path: /health
dockerfile: auto
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Agentfile"), []byte(content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')"), 0644))
}

func TestDryRun_NoExistingDeployment(t *testing.T) {
	dir := t.TempDir()
	writeDeployTestAgentfile(t, dir)

	mp := &testutil.MockPresenter{}
	err := runDryRun(dir, mp)
	require.NoError(t, err)

	found := false
	for _, r := range mp.ResultCalls {
		if r == "No existing deployment found. All files would be new:" {
			found = true
		}
	}
	assert.True(t, found, "should indicate all files are new")
}

func TestDryRun_NoChanges(t *testing.T) {
	dir := t.TempDir()
	writeDeployTestAgentfile(t, dir)

	// First generate artifacts normally
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	require.NoError(t, deploy.GenerateAll(af, tc, dir))

	// Now dry-run should show no changes
	mp := &testutil.MockPresenter{}
	err = runDryRun(dir, mp)
	require.NoError(t, err)

	found := false
	for _, r := range mp.ResultCalls {
		if r == "No changes detected" {
			found = true
		}
	}
	assert.True(t, found, "should report no changes")
}

func TestDryRun_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	writeDeployTestAgentfile(t, dir)

	// Generate artifacts
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	require.NoError(t, deploy.GenerateAll(af, tc, dir))

	// Modify an existing artifact
	composePath := filepath.Join(dir, deploy.OutputDir, "docker-compose.yml")
	require.NoError(t, os.WriteFile(composePath, []byte("# modified\n"), 0644))

	// Dry-run should detect changes
	mp := &testutil.MockPresenter{}
	err = runDryRun(dir, mp)
	require.NoError(t, err)

	assert.NotEmpty(t, mp.ResultCalls, "should show diff output")
}

func TestDryRun_NoAgentfile(t *testing.T) {
	dir := t.TempDir()

	mp := &testutil.MockPresenter{}
	err := runDryRun(dir, mp)
	require.Error(t, err)
}
