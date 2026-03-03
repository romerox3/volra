package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertGolden compares got against a golden file.
// When UPDATE_GOLDEN=1 is set, it overwrites the golden file with got.
func AssertGolden(t *testing.T, got string, goldenFile string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		err := os.WriteFile(goldenFile, []byte(got), 0644)
		require.NoError(t, err, "failed to write golden file %s", goldenFile)
		return
	}
	expected, err := os.ReadFile(goldenFile)
	require.NoError(t, err, "golden file %s not found — run with UPDATE_GOLDEN=1", goldenFile)
	assert.Equal(t, string(expected), got)
}
