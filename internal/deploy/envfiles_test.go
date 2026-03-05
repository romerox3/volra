package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateEnvFiles_AgentOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=secret123\nDB_URL=postgres://localhost\nUNUSED=foo\n"), 0644))

	af := &agentfile.Agentfile{
		Name: "test",
		Env:  []string{"API_KEY", "DB_URL"},
	}
	err := GenerateEnvFiles(af, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "agent.env"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "API_KEY=secret123")
	assert.Contains(t, string(content), "DB_URL=postgres://localhost")
	assert.NotContains(t, string(content), "UNUSED")
}

func TestGenerateEnvFiles_ServiceOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("POSTGRES_PASSWORD=pg123\nPOSTGRES_DB=mydb\n"), 0644))

	af := &agentfile.Agentfile{
		Name: "test",
		Services: map[string]agentfile.Service{
			"db": {Image: "postgres:16", Env: []string{"POSTGRES_PASSWORD", "POSTGRES_DB"}},
		},
	}
	err := GenerateEnvFiles(af, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, OutputDir, "test-db.env"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "POSTGRES_PASSWORD=pg123")
	assert.Contains(t, string(content), "POSTGRES_DB=mydb")
}

func TestGenerateEnvFiles_Separation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=secret\nPOSTGRES_PASSWORD=pg\n"), 0644))

	af := &agentfile.Agentfile{
		Name: "test",
		Env:  []string{"API_KEY"},
		Services: map[string]agentfile.Service{
			"db": {Image: "postgres:16", Env: []string{"POSTGRES_PASSWORD"}},
		},
	}
	err := GenerateEnvFiles(af, dir)
	require.NoError(t, err)

	// Agent env should NOT contain POSTGRES_PASSWORD
	agentContent, err := os.ReadFile(filepath.Join(dir, OutputDir, "agent.env"))
	require.NoError(t, err)
	assert.Contains(t, string(agentContent), "API_KEY=secret")
	assert.NotContains(t, string(agentContent), "POSTGRES_PASSWORD")

	// Service env should NOT contain API_KEY
	dbContent, err := os.ReadFile(filepath.Join(dir, OutputDir, "test-db.env"))
	require.NoError(t, err)
	assert.Contains(t, string(dbContent), "POSTGRES_PASSWORD=pg")
	assert.NotContains(t, string(dbContent), "API_KEY")
}

func TestGenerateEnvFiles_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=secret\n"), 0644))

	af := &agentfile.Agentfile{
		Name: "test",
		Env:  []string{"API_KEY"},
	}
	err := GenerateEnvFiles(af, dir)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, OutputDir, "agent.env"))
	require.NoError(t, err)
	// File should be 0600 (owner-only read/write)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestGenerateEnvFiles_NoEnvVars(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0644))

	af := &agentfile.Agentfile{
		Name: "test",
	}
	err := GenerateEnvFiles(af, dir)
	require.NoError(t, err)

	// No agent.env should be created
	_, err = os.Stat(filepath.Join(dir, OutputDir, "agent.env"))
	assert.True(t, os.IsNotExist(err))
}

func TestParseEnvFile_Comments(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("# Comment\nKEY=value\n\n# Another\nKEY2=val2\n"), 0644))

	m, err := parseEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "value", m["KEY"])
	assert.Equal(t, "val2", m["KEY2"])
	assert.Len(t, m, 2)
}
