package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppend_CreatesFileAndWritesEntry(t *testing.T) {
	dir := t.TempDir()

	entry := Entry{
		Action:     "deploy",
		Agent:      "my-agent",
		User:       "testuser",
		Result:     "success",
		DurationMs: 1234,
		Details:    map[string]any{"version": "0.7.0"},
	}
	require.NoError(t, Append(dir, entry))

	// Verify file exists.
	path := filepath.Join(dir, auditFile)
	assert.FileExists(t, path)

	// Read back.
	entries, err := Read(dir, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "deploy", entries[0].Action)
	assert.Equal(t, "my-agent", entries[0].Agent)
	assert.Equal(t, "testuser", entries[0].User)
	assert.Equal(t, "success", entries[0].Result)
	assert.Equal(t, int64(1234), entries[0].DurationMs)
	assert.NotZero(t, entries[0].Timestamp)
}

func TestAppend_AppendsMultipleEntries(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "a"}))
	require.NoError(t, Append(dir, Entry{Action: "down", Agent: "a"}))
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "b"}))

	entries, err := Read(dir, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestAppend_SetsTimestampAndUser(t *testing.T) {
	dir := t.TempDir()

	before := time.Now().UTC()
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "a"}))

	entries, err := Read(dir, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.False(t, entries[0].Timestamp.Before(before))
	// User should be set from $USER env.
	assert.Equal(t, os.Getenv("USER"), entries[0].User)
}

func TestRead_NoFileReturnsNil(t *testing.T) {
	dir := t.TempDir()

	entries, err := Read(dir, nil)
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestRead_FilterByAction(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "a"}))
	require.NoError(t, Append(dir, Entry{Action: "down", Agent: "a"}))
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "b"}))

	entries, err := Read(dir, &Filter{Action: "deploy"})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestRead_FilterByAgent(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "a"}))
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "b"}))

	entries, err := Read(dir, &Filter{Agent: "b"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "b", entries[0].Agent)
}

func TestRead_FilterBySince(t *testing.T) {
	dir := t.TempDir()

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "a", Timestamp: old}))
	require.NoError(t, Append(dir, Entry{Action: "deploy", Agent: "b"})) // now

	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entries, err := Read(dir, &Filter{Since: cutoff})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "b", entries[0].Agent)
}

func TestRead_SkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, auditFile)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	content := `{"action":"deploy","agent":"a","timestamp":"2026-03-09T00:00:00Z","user":"x","result":"ok","duration_ms":0}
not json at all
{"action":"down","agent":"b","timestamp":"2026-03-09T00:00:00Z","user":"y","result":"ok","duration_ms":0}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	entries, err := Read(dir, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}
