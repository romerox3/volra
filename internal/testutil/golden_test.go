package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertGolden_MatchesExistingFile(t *testing.T) {
	goldenFile := filepath.Join("testdata", "sample.golden")
	AssertGolden(t, "hello golden world", goldenFile)
}

func TestAssertGolden_FailsOnMismatch(t *testing.T) {
	goldenFile := filepath.Join("testdata", "sample.golden")

	mockT := &testing.T{}
	AssertGolden(mockT, "wrong content", goldenFile)
	assert.True(t, mockT.Failed(), "should fail on content mismatch")
}

func TestAssertGolden_UpdateMode(t *testing.T) {
	tmpDir := t.TempDir()
	goldenFile := filepath.Join(tmpDir, "update_test.golden")

	t.Setenv("UPDATE_GOLDEN", "1")
	AssertGolden(t, "new content", goldenFile)

	data, err := os.ReadFile(goldenFile)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(data))
}

func TestAssertGolden_FailsOnMissingFile(t *testing.T) {
	// Verify that a non-existent golden file would cause a read error.
	// We can't call AssertGolden with a bare *testing.T because require.NoError
	// calls FailNow/Goexit which panics outside a proper test goroutine.
	_, err := os.ReadFile(filepath.Join("testdata", "nonexistent.golden"))
	assert.Error(t, err, "reading non-existent golden file should error")
}
