package deploy

import (
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resolveHealthcheck tests ---

func TestResolveHealthcheck_PostgresExact(t *testing.T) {
	hc := resolveHealthcheck("postgres", nil)
	require.NotNil(t, hc)
	assert.Contains(t, hc.Test[1], "pg_isready")
}

func TestResolveHealthcheck_PostgresWithTag(t *testing.T) {
	hc := resolveHealthcheck("postgres:16-alpine", nil)
	require.NotNil(t, hc)
	assert.Contains(t, hc.Test[1], "pg_isready")
}

func TestResolveHealthcheck_Redis(t *testing.T) {
	hc := resolveHealthcheck("redis:7-alpine", nil)
	require.NotNil(t, hc)
	assert.Contains(t, hc.Test[1], "redis-cli")
}

func TestResolveHealthcheck_Chroma(t *testing.T) {
	// chromadb/chroma:latest is now a Rust binary without shell — no auto-healthcheck.
	hc := resolveHealthcheck("chromadb/chroma:0.4.24", nil)
	assert.Nil(t, hc, "chromadb removed from auto-healthcheck (Rust binary, no shell)")
}

func TestResolveHealthcheck_UnknownImage(t *testing.T) {
	hc := resolveHealthcheck("nginx:latest", nil)
	assert.Nil(t, hc)
}

func TestResolveHealthcheck_ExplicitOverride(t *testing.T) {
	explicit := &agentfile.HealthcheckConfig{
		Test:     []string{"CMD", "my-check"},
		Interval: "10s", Timeout: "5s", Retries: 3, StartPeriod: "20s",
	}
	hc := resolveHealthcheck("postgres:16", explicit)
	require.NotNil(t, hc)
	assert.Equal(t, "my-check", hc.Test[1])
	assert.Equal(t, "10s", hc.Interval)
}

func TestResolveHealthcheck_ExplicitWithoutCMDPrefix(t *testing.T) {
	explicit := &agentfile.HealthcheckConfig{
		Test:     []string{"redis-cli", "ping"},
		Interval: "10s", Timeout: "5s", Retries: 3,
	}
	hc := resolveHealthcheck("redis:7-alpine", explicit)
	require.NotNil(t, hc)
	assert.Equal(t, []string{"CMD", "redis-cli", "ping"}, hc.Test)
}

func TestEnsureCMDPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
	}{
		{"already CMD", []string{"CMD", "check"}, []string{"CMD", "check"}},
		{"already CMD-SHELL", []string{"CMD-SHELL", "check"}, []string{"CMD-SHELL", "check"}},
		{"already NONE", []string{"NONE"}, []string{"NONE"}},
		{"missing prefix", []string{"pg_isready", "-U", "cortex"}, []string{"CMD", "pg_isready", "-U", "cortex"}},
		{"empty", []string{}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, ensureCMDPrefix(tt.in))
		})
	}
}

// --- resolveResources tests ---

func TestResolveResources_Postgres(t *testing.T) {
	res := resolveResources("postgres:16", nil)
	require.NotNil(t, res)
	assert.Equal(t, "512m", res.MemLimit)
	assert.Equal(t, "0.5", res.CPUs)
}

func TestResolveResources_Redis(t *testing.T) {
	res := resolveResources("redis:7-alpine", nil)
	require.NotNil(t, res)
	assert.Equal(t, "256m", res.MemLimit)
}

func TestResolveResources_UnknownImage(t *testing.T) {
	res := resolveResources("nginx:latest", nil)
	assert.Nil(t, res)
}

func TestResolveResources_ExplicitOverride(t *testing.T) {
	explicit := &agentfile.ResourceConfig{
		MemLimit: "2g", CPUs: "2.0",
	}
	res := resolveResources("postgres:16", explicit)
	require.NotNil(t, res)
	assert.Equal(t, "2g", res.MemLimit)
	assert.Equal(t, "2.0", res.CPUs)
}

// --- imageMatchesPrefix tests ---

func TestImageMatchesPrefix_Exact(t *testing.T) {
	assert.True(t, imageMatchesPrefix("postgres", "postgres"))
}

func TestImageMatchesPrefix_WithTag(t *testing.T) {
	assert.True(t, imageMatchesPrefix("postgres:16", "postgres"))
}

func TestImageMatchesPrefix_WithOrg(t *testing.T) {
	assert.True(t, imageMatchesPrefix("chromadb/chroma:0.4.24", "chromadb/chroma"))
}

func TestImageMatchesPrefix_NoMatch(t *testing.T) {
	assert.False(t, imageMatchesPrefix("nginx:latest", "postgres"))
}

func TestImageMatchesPrefix_PartialNoMatch(t *testing.T) {
	// "postgresqldb" should NOT match "postgres" prefix
	assert.False(t, imageMatchesPrefix("postgresqldb", "postgres"))
}
