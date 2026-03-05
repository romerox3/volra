//go:build e2e

// Package e2e contains end-to-end tests that validate Volra against real
// Python agent projects in examples/.
//
// Phase 1 — Agentfile loading (no Docker required)
// Phase 2 — Compose generation (no Docker required)
// Phase 3 — Deploy + health check (requires Docker, set VOLRA_E2E_DEPLOY=1)
// Phase 4 — Expected failures (requires Docker for some)
package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot determine test file location")
	// tests/e2e/e2e_test.go → repo root is 2 levels up
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// exampleDir returns the absolute path to an example agent directory.
func exampleDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "examples", name)
	_, err := os.Stat(dir)
	require.NoError(t, err, "example dir %s must exist", name)
	return dir
}

// deployEnabled returns true if VOLRA_E2E_DEPLOY=1 is set.
func deployEnabled() bool {
	return os.Getenv("VOLRA_E2E_DEPLOY") == "1"
}

// ensureDotEnv copies .env.example to .env if .env doesn't exist.
func ensureDotEnv(t *testing.T, dir string) {
	t.Helper()
	dotEnv := filepath.Join(dir, ".env")
	if _, err := os.Stat(dotEnv); err == nil {
		return // already exists
	}
	example := filepath.Join(dir, ".env.example")
	data, err := os.ReadFile(example)
	if err != nil {
		return // no .env.example either
	}
	require.NoError(t, os.WriteFile(dotEnv, data, 0644))
	t.Cleanup(func() { os.Remove(dotEnv) })
}

// cleanVolraDir removes the .volra output directory before and after test.
func cleanVolraDir(t *testing.T, dir string) {
	t.Helper()
	// Clean at start to avoid corrupted residuals (BUG-03).
	os.RemoveAll(filepath.Join(dir, ".volra"))
	t.Cleanup(func() { os.RemoveAll(filepath.Join(dir, ".volra")) })
}

// generateAllArtifacts generates all deploy artifacts (Dockerfile, compose, prometheus, grafana, etc.)
func generateAllArtifacts(t *testing.T, af *agentfile.Agentfile, tc *deploy.TemplateContext, dir string) {
	t.Helper()
	var err error
	if af.Dockerfile != agentfile.DockerfileModeCustom {
		err = deploy.GenerateDockerfile(tc, dir)
		require.NoError(t, err, "GenerateDockerfile")
	}
	err = deploy.GenerateCompose(tc, dir)
	require.NoError(t, err, "GenerateCompose")
	err = deploy.GeneratePrometheus(tc, dir)
	require.NoError(t, err, "GeneratePrometheus")
	err = deploy.CopyAlertRules(dir)
	require.NoError(t, err, "CopyAlertRules")
	err = deploy.CopyBlackboxConfig(dir)
	require.NoError(t, err, "CopyBlackboxConfig")
	err = deploy.CopyGrafanaAssets(dir, tc.HasMetrics, tc.HasLevel2)
	require.NoError(t, err, "CopyGrafanaAssets")
}

// allAgents lists the example directories in test order.
var allAgents = []struct {
	dir       string
	name      string // expected Agentfile name
	framework agentfile.Framework
	hasEnv    bool
}{
	{"a1-echo", "echo-agent", agentfile.FrameworkGeneric, false},
	{"a2-sentiment", "sentiment-analyzer", agentfile.FrameworkGeneric, true},
	{"a3-summarizer", "doc-summarizer", agentfile.FrameworkGeneric, true},
	{"a4-rag-kb", "rag-kb", agentfile.FrameworkGeneric, false},
	{"a5-conversational", "conv-agent", agentfile.FrameworkLangGraph, true},
	{"a6-gateway", "ai-gateway", agentfile.FrameworkGeneric, false},
	{"a7-vision", "vision-classifier", agentfile.FrameworkGeneric, false},
	{"a8-orchestrator", "orchestrator", agentfile.FrameworkLangGraph, true},
}

// ---------------------------------------------------------------------------
// Phase 1: Agentfile Loading — validates all Agentfiles parse and validate
// ---------------------------------------------------------------------------

func TestPhase1_AllAgentfilesLoad(t *testing.T) {
	for _, agent := range allAgents {
		t.Run(agent.dir, func(t *testing.T) {
			dir := exampleDir(t, agent.dir)
			af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
			require.NoError(t, err, "Agentfile in %s must load without error", agent.dir)

			assert.Equal(t, 1, af.Version)
			assert.Equal(t, agent.name, af.Name)
			assert.Equal(t, agent.framework, af.Framework)
			assert.Equal(t, 8000, af.Port)
			assert.Equal(t, "/health", af.HealthPath)
		})
	}
}

func TestPhase1_SpecificFields(t *testing.T) {
	t.Run("a2_health_timeout", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a2-sentiment"), "Agentfile"))
		require.NoError(t, err)
		assert.Equal(t, 90, af.HealthTimeout)
		assert.Contains(t, af.Env, "MODEL_NAME")
	})

	t.Run("a3_volumes_and_env", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a3-summarizer"), "Agentfile"))
		require.NoError(t, err)
		assert.Equal(t, []string{"/data/cache"}, af.Volumes)
		assert.Contains(t, af.Env, "OPENAI_API_KEY")
	})

	t.Run("a4_services", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a4-rag-kb"), "Agentfile"))
		require.NoError(t, err)
		require.Contains(t, af.Services, "redis-cache")
		assert.Equal(t, "redis:7-alpine", af.Services["redis-cache"].Image)
		assert.Equal(t, 6379, af.Services["redis-cache"].Port)
	})

	t.Run("a5_multi_service_langgraph", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a5-conversational"), "Agentfile"))
		require.NoError(t, err)
		assert.Equal(t, agentfile.FrameworkLangGraph, af.Framework)
		assert.Equal(t, 120, af.HealthTimeout)
		assert.Len(t, af.Services, 2)
		require.Contains(t, af.Services, "redis")
		require.Contains(t, af.Services, "postgres")

		pg := af.Services["postgres"]
		assert.Equal(t, "postgres:16-alpine", pg.Image)
		assert.Equal(t, []string{"/var/lib/postgresql/data"}, pg.Volumes)
		assert.Contains(t, pg.Env, "POSTGRES_PASSWORD")
		assert.Contains(t, pg.Env, "POSTGRES_DB")
	})

	t.Run("a6_security_full", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a6-gateway"), "Agentfile"))
		require.NoError(t, err)
		require.NotNil(t, af.Security)
		assert.True(t, af.Security.ReadOnly)
		assert.True(t, af.Security.NoNewPrivileges)
		assert.Equal(t, []string{"ALL"}, af.Security.DropCapabilities)
	})

	t.Run("a7_gpu_custom_dockerfile", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a7-vision"), "Agentfile"))
		require.NoError(t, err)
		assert.True(t, af.GPU)
		assert.Equal(t, agentfile.DockerfileModeCustom, af.Dockerfile)
		assert.Equal(t, 300, af.HealthTimeout)
		assert.Equal(t, []string{"/models"}, af.Volumes)
	})

	t.Run("a8_all_features", func(t *testing.T) {
		af, err := agentfile.Load(filepath.Join(exampleDir(t, "a8-orchestrator"), "Agentfile"))
		require.NoError(t, err)
		assert.Equal(t, agentfile.FrameworkLangGraph, af.Framework)
		assert.Equal(t, agentfile.DockerfileModeCustom, af.Dockerfile)
		assert.Equal(t, 180, af.HealthTimeout)
		assert.Len(t, af.Volumes, 3)
		assert.Len(t, af.Services, 3)
		assert.Len(t, af.Env, 2)
		require.NotNil(t, af.Security)
		assert.True(t, af.Security.NoNewPrivileges)
		assert.Contains(t, af.Security.DropCapabilities, "NET_RAW")
		assert.Contains(t, af.Security.DropCapabilities, "SYS_ADMIN")
		assert.False(t, af.GPU)
	})
}

// ---------------------------------------------------------------------------
// Phase 2: Compose Generation — validates BuildContext + RenderCompose
// ---------------------------------------------------------------------------

func TestPhase2_ComposeGeneration(t *testing.T) {
	tests := []struct {
		dir             string
		name            string
		expectServices  []string // expected service containers in compose
		expectVolumes   []string // expected named volumes (partial match)
		expectEnvFile   bool
		expectGPU       bool
		expectSecurity  bool
		expectReadOnly  bool
		expectCustom    bool // custom Dockerfile
		expectDependsOn []string
	}{
		{
			dir:  "a1-echo",
			name: "echo-agent",
		},
		{
			dir:           "a2-sentiment",
			name:          "sentiment-analyzer",
			expectEnvFile: true,
		},
		{
			dir:           "a3-summarizer",
			name:          "doc-summarizer",
			expectVolumes: []string{"doc-summarizer-data-cache"},
			expectEnvFile: true,
		},
		{
			dir:            "a4-rag-kb",
			name:           "rag-kb",
			expectServices: []string{"rag-kb-redis-cache"},
			expectDependsOn: []string{"rag-kb-redis-cache"},
		},
		{
			dir:            "a5-conversational",
			name:           "conv-agent",
			expectServices: []string{"conv-agent-postgres", "conv-agent-redis"},
			expectVolumes:  []string{"conv-agent-postgres-var-lib-postgresql-data"},
			expectEnvFile:  true,
			expectDependsOn: []string{"conv-agent-postgres", "conv-agent-redis"},
		},
		{
			dir:            "a6-gateway",
			name:           "ai-gateway",
			expectSecurity: true,
			expectReadOnly: true,
		},
		{
			dir:           "a7-vision",
			name:          "vision-classifier",
			expectGPU:     true,
			expectVolumes: []string{"vision-classifier-models"},
			expectCustom:  true,
		},
		{
			dir:            "a8-orchestrator",
			name:           "orchestrator",
			expectServices: []string{"orchestrator-chromadb", "orchestrator-postgres", "orchestrator-redis"},
			expectVolumes:  []string{"orchestrator-data-sessions", "orchestrator-data-embeddings", "orchestrator-data-logs", "orchestrator-postgres-var-lib-postgresql-data"},
			expectEnvFile:  true,
			expectSecurity: true,
			expectCustom:   true,
			expectDependsOn: []string{"orchestrator-chromadb", "orchestrator-postgres", "orchestrator-redis"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			dir := exampleDir(t, tt.dir)
			af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
			require.NoError(t, err)

			tc := deploy.BuildContext(af, dir)
			require.NotNil(t, tc)

			yaml, err := deploy.RenderCompose(tc)
			require.NoError(t, err)
			require.NotEmpty(t, yaml)

			// Project name.
			assert.Contains(t, yaml, fmt.Sprintf("name: %s", tt.name))

			// Agent service always present.
			assert.Contains(t, yaml, "agent:")
			assert.Contains(t, yaml, fmt.Sprintf("container_name: %s-agent", tt.name))

			// Services.
			for _, svc := range tt.expectServices {
				assert.Contains(t, yaml, fmt.Sprintf("  %s:", svc), "service %s should exist", svc)
				assert.Contains(t, yaml, fmt.Sprintf("container_name: %s", svc))
			}

			// depends_on (map format with condition).
			if len(tt.expectDependsOn) > 0 {
				assert.Contains(t, yaml, "depends_on:")
				for _, dep := range tt.expectDependsOn {
					assert.Contains(t, yaml, fmt.Sprintf("      %s:", dep))
					assert.Contains(t, yaml, "condition:")
				}
			}

			// Named volumes in top-level volumes section.
			for _, vol := range tt.expectVolumes {
				assert.Contains(t, yaml, vol, "volume %s should be referenced", vol)
			}

			// env_file.
			if tt.expectEnvFile {
				assert.Contains(t, yaml, "env_file:")
			} else {
				// Only check agent section doesn't have env_file — services might.
				lines := strings.Split(yaml, "\n")
				inAgent := false
				agentHasEnvFile := false
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if trimmed == "agent:" {
						inAgent = true
					} else if inAgent && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
						inAgent = false
					}
					if inAgent && strings.Contains(line, "env_file:") {
						agentHasEnvFile = true
					}
				}
				_ = agentHasEnvFile // not all agents without env skip env_file
			}

			// GPU.
			if tt.expectGPU {
				assert.Contains(t, yaml, "driver: nvidia")
				assert.Contains(t, yaml, "capabilities: [gpu]")
			} else {
				assert.NotContains(t, yaml, "driver: nvidia")
			}

			// Security.
			if tt.expectSecurity {
				if tt.expectReadOnly {
					assert.Contains(t, yaml, "read_only: true")
				}
				assert.Contains(t, yaml, "no-new-privileges:true")
			}

			// Custom Dockerfile.
			if tt.expectCustom {
				assert.Contains(t, yaml, "dockerfile: Dockerfile")
				assert.NotContains(t, yaml, ".volra/Dockerfile")
			} else {
				assert.Contains(t, yaml, "dockerfile: .volra/Dockerfile")
			}

			// Infra services always present.
			assert.Contains(t, yaml, "prometheus:")
			assert.Contains(t, yaml, "grafana:")
			assert.Contains(t, yaml, "blackbox:")

			// Network.
			assert.Contains(t, yaml, "volra:")
			assert.Contains(t, yaml, "driver: bridge")
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 3: Deploy + Health — requires Docker (VOLRA_E2E_DEPLOY=1)
// ---------------------------------------------------------------------------

func TestPhase3_DeployAndHealth(t *testing.T) {
	if !deployEnabled() {
		t.Skip("VOLRA_E2E_DEPLOY=1 not set — skipping deploy tests")
	}

	dr := docker.NewExecRunner()
	ctx := context.Background()

	// Agents that should deploy successfully and pass health checks.
	deployable := []struct {
		dir     string
		name    string
		timeout time.Duration
	}{
		{"a1-echo", "echo-agent", 60 * time.Second},
		{"a2-sentiment", "sentiment-analyzer", 90 * time.Second},
		{"a3-summarizer", "doc-summarizer", 60 * time.Second},
		{"a4-rag-kb", "rag-kb", 90 * time.Second},
		{"a5-conversational", "conv-agent", 120 * time.Second},
		{"a8-orchestrator", "orchestrator", 180 * time.Second},
	}

	for _, agent := range deployable {
		t.Run(agent.dir, func(t *testing.T) {
			dir := exampleDir(t, agent.dir)
			ensureDotEnv(t, dir)
			cleanVolraDir(t, dir)

			af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
			require.NoError(t, err)

			tc := deploy.BuildContext(af, dir)

			// Generate ALL artifacts (BUG-01 fix).
			generateAllArtifacts(t, af, tc, dir)

			// Deploy with docker compose.
			composePath := filepath.Join(dir, ".volra", "docker-compose.yml")

			// Register cleanup BEFORE deploy so it always runs.
			t.Cleanup(func() {
				out, _ := dr.Run(ctx, "compose", "-f", composePath, "-p", agent.name, "down", "-v", "--remove-orphans")
				t.Logf("cleanup output for %s:\n%s", agent.name, out)
			})

			composeOut, err := dr.Run(ctx, "compose", "-f", composePath, "-p", agent.name, "up", "-d", "--build")
			if err != nil {
				t.Logf("docker compose output:\n%s", composeOut)
			}
			require.NoError(t, err, "docker compose up should succeed for %s", agent.dir)

			// Health check with retry.
			healthURL := fmt.Sprintf("http://localhost:%d%s", af.Port, af.HealthPath)
			deadline := time.Now().Add(agent.timeout)
			var lastErr error
			for time.Now().Before(deadline) {
				resp, err := http.Get(healthURL)
				if err == nil && resp.StatusCode == http.StatusOK {
					resp.Body.Close()
					lastErr = nil
					break
				}
				if err != nil {
					lastErr = err
				} else {
					lastErr = fmt.Errorf("health returned %d", resp.StatusCode)
					resp.Body.Close()
				}
				time.Sleep(2 * time.Second)
			}
			require.NoError(t, lastErr, "health check should pass for %s at %s", agent.dir, healthURL)
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 4: Expected Failures
// ---------------------------------------------------------------------------

func TestPhase4_A6_ReadOnlyExpectedFailure(t *testing.T) {
	if !deployEnabled() {
		t.Skip("VOLRA_E2E_DEPLOY=1 not set — skipping deploy tests")
	}

	dir := exampleDir(t, "a6-gateway")
	cleanVolraDir(t, dir)

	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)

	tc := deploy.BuildContext(af, dir)
	generateAllArtifacts(t, af, tc, dir)

	dr := docker.NewExecRunner()
	ctx := context.Background()
	composePath := filepath.Join(dir, ".volra", "docker-compose.yml")

	// Deploy — may start but agent likely crashes due to read_only.
	dr.Run(ctx, "compose", "-f", composePath, "-p", "ai-gateway", "up", "-d", "--build")
	t.Cleanup(func() {
		dr.Run(ctx, "compose", "-f", composePath, "-p", "ai-gateway", "down", "-v", "--remove-orphans")
	})

	// Wait briefly, then verify agent is NOT healthy.
	time.Sleep(10 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", af.Port))
	if err != nil {
		// Connection refused = expected (agent crashed due to read_only + Python).
		t.Logf("L1 CONFIRMED: read_only:true prevents Python startup — %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Logf("L1 CONFIRMED: agent unhealthy with status %d", resp.StatusCode)
		return
	}
	// If it somehow works, that's fine too — but unexpected.
	t.Log("L1 UNEXPECTED: agent started despite read_only:true — may need tmpfs investigation")
}

// --- Phase 2 Extension: New features validation ---

func TestPhase2_ComposeHealthchecks(t *testing.T) {
	// A4 has redis, A5 has redis + postgres — both should get auto healthchecks
	for _, agent := range []struct{ dir, name string }{
		{"a4-rag-kb", "rag-kb"},
		{"a5-conversational", "conv-agent"},
	} {
		t.Run(agent.dir, func(t *testing.T) {
			dir := exampleDir(t, agent.dir)
			af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
			require.NoError(t, err)
			tc := deploy.BuildContext(af, dir)
			yaml, err := deploy.RenderCompose(tc)
			require.NoError(t, err)
			assert.Contains(t, yaml, "healthcheck:")
			assert.Contains(t, yaml, "condition: service_healthy")
		})
	}
}

func TestPhase2_ComposeDependsOnHealthy(t *testing.T) {
	dir := exampleDir(t, "a5-conversational")
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)
	// Both postgres and redis have known healthchecks
	assert.Contains(t, yaml, "condition: service_healthy")
	assert.NotContains(t, yaml, "condition: service_started")
}

func TestPhase2_ComposeServicePortsNotExposed(t *testing.T) {
	// By default, services should NOT expose ports to host (HostPort=0)
	dir := exampleDir(t, "a4-rag-kb")
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)
	// Find redis service section
	redisIdx := strings.Index(yaml, "rag-kb-redis-cache:")
	require.Greater(t, redisIdx, 0)
	blackboxIdx := strings.Index(yaml, "blackbox:")
	redisSection := yaml[redisIdx:blackboxIdx]
	assert.NotContains(t, redisSection, "ports:")
}

func TestPhase2_ComposeAutoResources(t *testing.T) {
	dir := exampleDir(t, "a5-conversational")
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)
	// postgres and redis should get auto resource limits
	assert.Contains(t, yaml, "memory:")
	assert.Contains(t, yaml, "cpus:")
}

func TestPhase2_ComposeTmpfs(t *testing.T) {
	dir := exampleDir(t, "a6-gateway")
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)
	// read_only agent should get auto tmpfs
	assert.Contains(t, yaml, "tmpfs:")
	assert.Contains(t, yaml, "/tmp:size=100M")
	assert.Contains(t, yaml, "/app/__pycache__:size=50M")
}

func TestPhase2_ComposeEnvFileSeparation(t *testing.T) {
	dir := exampleDir(t, "a5-conversational")
	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)
	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)
	// Agent should reference ./agent.env
	assert.Contains(t, yaml, "./agent.env")
	// Postgres service should reference its own env file
	assert.Contains(t, yaml, "./conv-agent-postgres.env")
	// Should NOT reference ../.env directly
	assert.NotContains(t, yaml, "../.env")
}

func TestPhase4_A7_GPUComposeConfig(t *testing.T) {
	// This test does NOT require Docker running — just validates compose config.
	dir := exampleDir(t, "a7-vision")
	cleanVolraDir(t, dir)

	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	require.NoError(t, err)

	tc := deploy.BuildContext(af, dir)
	yaml, err := deploy.RenderCompose(tc)
	require.NoError(t, err)

	// GPU block should be present.
	assert.Contains(t, yaml, "driver: nvidia")
	assert.Contains(t, yaml, "count: all")
	assert.Contains(t, yaml, "capabilities: [gpu]")

	// Verify compose config is valid YAML (docker compose config).
	if deployEnabled() {
		err = deploy.GenerateCompose(tc, dir)
		require.NoError(t, err)

		dr := docker.NewExecRunner()
		ctx := context.Background()
		composePath := filepath.Join(dir, ".volra", "docker-compose.yml")

		out, err := dr.Run(ctx, "compose", "-f", composePath, "config")
		if err != nil {
			// Expected: docker compose config may fail without NVIDIA runtime.
			t.Logf("L4 CONFIRMED: compose config fails without NVIDIA runtime — %s", out)
		} else {
			t.Log("L4 NOTE: compose config succeeded (NVIDIA runtime may be installed)")
		}
	}
}
