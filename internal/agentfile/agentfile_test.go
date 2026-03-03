package agentfile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antonioromero/volra/internal/agentfile"
	"github.com/antonioromero/volra/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Parse tests ---

func TestParse_ValidFull(t *testing.T) {
	r := openFixture(t, "valid_full.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)

	assert.Equal(t, 1, af.Version)
	assert.Equal(t, "my-agent", af.Name)
	assert.Equal(t, agentfile.FrameworkLangGraph, af.Framework)
	assert.Equal(t, 9000, af.Port)
	assert.Equal(t, "/healthz", af.HealthPath)
	assert.Equal(t, []string{"OPENAI_API_KEY", "DATABASE_URL"}, af.Env)
	assert.Equal(t, agentfile.DockerfileModeCustom, af.Dockerfile)
}

func TestParse_ValidMinimal(t *testing.T) {
	r := openFixture(t, "valid_minimal.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)

	assert.Equal(t, 1, af.Version)
	assert.Equal(t, "my-agent", af.Name)
	assert.Equal(t, agentfile.FrameworkGeneric, af.Framework)
	assert.Equal(t, 8000, af.Port)
	assert.Equal(t, "/health", af.HealthPath)
	assert.Empty(t, af.Env)
	assert.Equal(t, agentfile.DockerfileModeAuto, af.Dockerfile)
}

func TestParse_InvalidFramework(t *testing.T) {
	r := openFixture(t, "invalid_framework.yaml")
	_, err := agentfile.Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "framework")
}

func TestParse_InvalidDockerfile(t *testing.T) {
	r := openFixture(t, "invalid_dockerfile.yaml")
	_, err := agentfile.Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dockerfile")
}

func TestParse_BadYAML(t *testing.T) {
	r := openFixture(t, "bad_yaml.yaml")
	_, err := agentfile.Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing Agentfile")
}

func TestParse_UnknownField(t *testing.T) {
	r := openFixture(t, "unknown_field.yaml")
	_, err := agentfile.Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown_field")
}

func TestParse_FromString(t *testing.T) {
	yaml := `version: 1
name: test-agent
framework: generic
port: 8080
health_path: /ready
dockerfile: auto
`
	af, err := agentfile.Parse(strings.NewReader(yaml))
	require.NoError(t, err)
	assert.Equal(t, "test-agent", af.Name)
	assert.Equal(t, 8080, af.Port)
	assert.Equal(t, "/ready", af.HealthPath)
}

// --- Validate tests ---

func TestValidate_ValidAgentfile(t *testing.T) {
	af := validAgentfile()
	err := agentfile.Validate(af)
	assert.NoError(t, err)
}

func TestValidate_MissingVersion(t *testing.T) {
	af := validAgentfile()
	af.Version = 0
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "version")
}

func TestValidate_VersionTooNew(t *testing.T) {
	af := validAgentfile()
	af.Version = 2
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeUnsupportedVersion, "version 2")
}

func TestValidate_MissingName(t *testing.T) {
	af := validAgentfile()
	af.Name = ""
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "name")
}

func TestValidate_NameTooShort(t *testing.T) {
	af := validAgentfile()
	af.Name = "a"
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "name")
}

func TestValidate_NameTooLong(t *testing.T) {
	af := validAgentfile()
	af.Name = "a" + strings.Repeat("b", 62) + "c" // 64 chars
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "name")
}

func TestValidate_NameNotDNSLabel(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"uppercase", "MyAgent"},
		{"spaces", "my agent"},
		{"special chars", "my-agent!"},
		{"starts with digit", "1agent"},
		{"starts with hyphen", "-agent"},
		{"ends with hyphen", "agent-"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			af := validAgentfile()
			af.Name = tc.val
			err := agentfile.Validate(af)
			requireUserError(t, err, output.CodeInvalidAgentfile, "name")
		})
	}
}

func TestValidate_ValidDNSNames(t *testing.T) {
	cases := []string{
		"ab",               // min length
		"my-agent",         // typical
		"agent-v2",         // ends with digit
		"a-b-c-d",          // multiple hyphens
		"my-long-agent-99", // longer name
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			af := validAgentfile()
			af.Name = name
			err := agentfile.Validate(af)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_MissingPort(t *testing.T) {
	af := validAgentfile()
	af.Port = 0
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "port")
}

func TestValidate_PortOutOfRange(t *testing.T) {
	cases := []int{-1, 65536, 99999}
	for _, port := range cases {
		t.Run("port_"+strings.ReplaceAll(fmt.Sprint(port), "-", "neg"), func(t *testing.T) {
			af := validAgentfile()
			af.Port = port
			err := agentfile.Validate(af)
			requireUserError(t, err, output.CodeInvalidAgentfile, "port")
		})
	}
}

func TestValidate_ValidPorts(t *testing.T) {
	cases := []int{1, 80, 8000, 8080, 65535}
	for _, port := range cases {
		t.Run(fmt.Sprintf("port_%d", port), func(t *testing.T) {
			af := validAgentfile()
			af.Port = port
			err := agentfile.Validate(af)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_MissingHealthPath(t *testing.T) {
	af := validAgentfile()
	af.HealthPath = ""
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_path")
}

func TestValidate_HealthPathNoSlash(t *testing.T) {
	af := validAgentfile()
	af.HealthPath = "health"
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_path")
}

func TestValidate_EmptyEnvEntry(t *testing.T) {
	af := validAgentfile()
	af.Env = []string{"VALID", ""}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "env")
}

func TestValidate_DuplicateEnv(t *testing.T) {
	af := validAgentfile()
	af.Env = []string{"FOO", "BAR", "FOO"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "env")
}

func TestValidate_EmptyEnvListOK(t *testing.T) {
	af := validAgentfile()
	af.Env = nil
	err := agentfile.Validate(af)
	assert.NoError(t, err)
}

// --- Load tests ---

func TestLoad_ValidFull(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_full.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "my-agent", af.Name)
	assert.Equal(t, agentfile.FrameworkLangGraph, af.Framework)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := agentfile.Load("/nonexistent/Agentfile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opening Agentfile")
}

func TestLoad_ParseError(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_framework.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "framework")
}

func TestLoad_ValidationError(t *testing.T) {
	_, err := agentfile.Load(fixturePath("missing_version.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "version")
}

func TestLoad_VersionTooNew(t *testing.T) {
	_, err := agentfile.Load(fixturePath("version_too_new.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeUnsupportedVersion, "version 2")
}

func TestLoad_InvalidPort(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_port.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "port")
}

func TestLoad_InvalidHealthPath(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_health_path.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_path")
}

func TestLoad_DuplicateEnv(t *testing.T) {
	_, err := agentfile.Load(fixturePath("duplicate_env.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "env")
}

func TestLoad_EmptyEnvEntry(t *testing.T) {
	_, err := agentfile.Load(fixturePath("empty_env_entry.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "env")
}

func TestLoad_InvalidNameDNS(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_name_dns.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "name")
}

func TestLoad_UnknownField(t *testing.T) {
	_, err := agentfile.Load(fixturePath("unknown_field.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown_field")
}

// --- Framework type tests ---

func TestFramework_UnmarshalYAML_Valid(t *testing.T) {
	cases := []struct {
		input    string
		expected agentfile.Framework
	}{
		{"generic", agentfile.FrameworkGeneric},
		{"langgraph", agentfile.FrameworkLangGraph},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			yaml := fmt.Sprintf("version: 1\nname: ab\nframework: %s\nport: 8000\nhealth_path: /h\ndockerfile: auto\n", tc.input)
			af, err := agentfile.Parse(strings.NewReader(yaml))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, af.Framework)
		})
	}
}

func TestDockerfileMode_UnmarshalYAML_Valid(t *testing.T) {
	cases := []struct {
		input    string
		expected agentfile.DockerfileMode
	}{
		{"auto", agentfile.DockerfileModeAuto},
		{"custom", agentfile.DockerfileModeCustom},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			yaml := fmt.Sprintf("version: 1\nname: ab\nframework: generic\nport: 8000\nhealth_path: /h\ndockerfile: %s\n", tc.input)
			af, err := agentfile.Parse(strings.NewReader(yaml))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, af.Dockerfile)
		})
	}
}

// --- Helpers ---

func validAgentfile() *agentfile.Agentfile {
	return &agentfile.Agentfile{
		Version:    1,
		Name:       "my-agent",
		Framework:  agentfile.FrameworkGeneric,
		Port:       8000,
		HealthPath: "/health",
		Dockerfile: agentfile.DockerfileModeAuto,
	}
}

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(fixturePath(name))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func requireUserError(t *testing.T, err error, code string, containsWhat string) {
	t.Helper()
	require.Error(t, err)
	ue, ok := err.(*output.UserError)
	require.True(t, ok, "expected *output.UserError, got %T: %v", err, err)
	assert.Equal(t, code, ue.Code)
	assert.Contains(t, ue.What, containsWhat)
}
