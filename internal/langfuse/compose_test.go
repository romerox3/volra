package langfuse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCompose(t *testing.T) {
	cfg := Config{
		Port:         3100,
		PostgresPort: 5433,
		AgentName:    "my-agent",
	}

	result, err := RenderCompose(cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "langfuse/langfuse:2")
	assert.Contains(t, result, "postgres:16-alpine")
	assert.Contains(t, result, `"3100:3000"`)
	assert.Contains(t, result, `"5433:5432"`)
	assert.Contains(t, result, "my-agent")
	assert.Contains(t, result, "langfuse-db-data")
}

func TestRenderCompose_Defaults(t *testing.T) {
	cfg := Config{AgentName: "test"}

	result, err := RenderCompose(cfg)
	require.NoError(t, err)

	assert.Contains(t, result, `"3100:3000"`)
	assert.Contains(t, result, `"5433:5432"`)
}

func TestDashboardURL(t *testing.T) {
	assert.Equal(t, "http://localhost:3100", DashboardURL(0))
	assert.Equal(t, "http://localhost:4000", DashboardURL(4000))
}
