package otel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWrapper(t *testing.T) {
	cfg := WrapperConfig{
		OriginalEntrypoint: "python main.py",
		ServiceName:        "my-agent",
		CollectorEndpoint:  "http://otel-collector:4318",
	}

	result, err := RenderWrapper(cfg)
	require.NoError(t, err)

	assert.Contains(t, result, `OTEL_SERVICE_NAME="my-agent"`)
	assert.Contains(t, result, `OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4318"`)
	assert.Contains(t, result, "exec opentelemetry-instrument python main.py")
	assert.True(t, strings.HasPrefix(result, "#!/bin/sh"), "should start with shebang")
}

func TestRenderCollectorConfig(t *testing.T) {
	cfg := CollectorConfig{
		PrometheusPort: 8889,
		Agents:         []string{"agent-a", "agent-b"},
	}

	result, err := RenderCollectorConfig(cfg)
	require.NoError(t, err)

	assert.Contains(t, result, "endpoint: \"0.0.0.0:4318\"")
	assert.Contains(t, result, "endpoint: \"0.0.0.0:8889\"")
	assert.Contains(t, result, "namespace: volra")
}

func TestRenderCollectorConfig_DefaultPort(t *testing.T) {
	cfg := CollectorConfig{}

	result, err := RenderCollectorConfig(cfg)
	require.NoError(t, err)
	assert.Contains(t, result, "endpoint: \"0.0.0.0:8889\"")
}

func TestComposeService(t *testing.T) {
	svc := ComposeService(8889)

	assert.Equal(t, "otel/opentelemetry-collector-contrib:0.98.0", svc["image"])
	assert.Equal(t, "unless-stopped", svc["restart"])

	ports := svc["ports"].([]string)
	assert.Contains(t, ports, "4318:4318")
	assert.Contains(t, ports, "8889:8889")
}

func TestComposeService_DefaultPort(t *testing.T) {
	svc := ComposeService(0)
	ports := svc["ports"].([]string)
	assert.Contains(t, ports, "8889:8889")
}

func TestDockerfileSnippet(t *testing.T) {
	snippet := DockerfileSnippet("python main.py")

	assert.Contains(t, snippet, "COPY .volra/otel-wrapper.sh")
	assert.Contains(t, snippet, "opentelemetry-distro")
	assert.Contains(t, snippet, "ENTRYPOINT")
}
