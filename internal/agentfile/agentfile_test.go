package agentfile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/output"
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
	assert.Equal(t, 0, af.HealthTimeout) // not specified
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

// --- Health timeout tests ---

func TestValidate_HealthTimeoutZero(t *testing.T) {
	af := validAgentfile()
	af.HealthTimeout = 0
	err := agentfile.Validate(af)
	assert.NoError(t, err) // zero means default
}

func TestValidate_HealthTimeoutValid(t *testing.T) {
	cases := []int{10, 60, 120, 300, 600}
	for _, timeout := range cases {
		t.Run(fmt.Sprintf("timeout_%d", timeout), func(t *testing.T) {
			af := validAgentfile()
			af.HealthTimeout = timeout
			err := agentfile.Validate(af)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_HealthTimeoutTooLow(t *testing.T) {
	cases := []int{1, 5, 9}
	for _, timeout := range cases {
		t.Run(fmt.Sprintf("timeout_%d", timeout), func(t *testing.T) {
			af := validAgentfile()
			af.HealthTimeout = timeout
			err := agentfile.Validate(af)
			requireUserError(t, err, output.CodeInvalidAgentfile, "health_timeout")
		})
	}
}

func TestValidate_HealthTimeoutTooHigh(t *testing.T) {
	af := validAgentfile()
	af.HealthTimeout = 700
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_timeout")
}

func TestParse_ValidHealthTimeout(t *testing.T) {
	r := openFixture(t, "valid_health_timeout.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	assert.Equal(t, 300, af.HealthTimeout)
}

func TestLoad_InvalidHealthTimeoutLow(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_health_timeout_low.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_timeout")
}

func TestLoad_InvalidHealthTimeoutHigh(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_health_timeout_high.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "health_timeout")
}

func TestLoad_BackwardCompatNoHealthTimeout(t *testing.T) {
	// v1.0 Agentfile without health_timeout should still work
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 0, af.HealthTimeout) // defaults to zero
}

// --- Volumes tests ---

func TestValidate_VolumesValid(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/data", "/models"}
	err := agentfile.Validate(af)
	assert.NoError(t, err)
}

func TestValidate_VolumesEmpty(t *testing.T) {
	af := validAgentfile()
	af.Volumes = nil
	err := agentfile.Validate(af)
	assert.NoError(t, err)
}

func TestValidate_VolumesNotAbsolute(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"data"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesAppPath(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/app/data"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesAppExact(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/app"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesRootPath(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesDuplicate(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/data", "/data"}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesEmptyEntry(t *testing.T) {
	af := validAgentfile()
	af.Volumes = []string{"/data", ""}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestValidate_VolumesTooMany(t *testing.T) {
	af := validAgentfile()
	af.Volumes = make([]string, 11)
	for i := range af.Volumes {
		af.Volumes[i] = fmt.Sprintf("/vol%d", i)
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestParse_ValidVolumes(t *testing.T) {
	r := openFixture(t, "valid_volumes.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"/data", "/models"}, af.Volumes)
}

func TestLoad_InvalidVolumesNotAbsolute(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_volumes_not_absolute.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestLoad_InvalidVolumesAppPath(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_volumes_app_path.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestLoad_InvalidVolumesDuplicate(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_volumes_duplicate.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "volumes")
}

func TestLoad_BackwardCompatNoVolumes(t *testing.T) {
	// v1.0/v1.1 Agentfile without volumes should still work
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Volumes)
}

// --- Service validation tests ---

func TestValidate_ServicesValid(t *testing.T) {
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"redis": {Image: "redis:7-alpine"},
		"db":    {Image: "postgres:16", Port: 5432, Env: []string{"POSTGRES_PASSWORD"}},
	}
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_ServicesEmpty(t *testing.T) {
	af := validAgentfile()
	af.Services = nil
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_ServicesReservedName(t *testing.T) {
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"agent": {Image: "redis:7"},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "reserved")
}

func TestValidate_ServicesNoImage(t *testing.T) {
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"redis": {Port: 6379},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "image")
}

func TestValidate_ServicesPortConflict(t *testing.T) {
	// Container ports don't conflict (each container has its own namespace).
	// Only host_ports can conflict with the agent port.
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"redis": {Image: "redis:7", Port: 8000}, // same container port as agent — OK
	}
	err := agentfile.Validate(af)
	assert.NoError(t, err, "container ports should not conflict between services")

	// But host_port conflict with agent port should still be an error.
	af2 := validAgentfile()
	af2.Services = map[string]agentfile.Service{
		"redis": {Image: "redis:7", Port: 6379, HostPort: 8000},
	}
	err2 := agentfile.Validate(af2)
	requireUserError(t, err2, output.CodeInvalidAgentfile, "host_port")
}

func TestValidate_ServicesTooMany(t *testing.T) {
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"svc-a": {Image: "a"}, "svc-b": {Image: "b"}, "svc-c": {Image: "c"},
		"svc-d": {Image: "d"}, "svc-e": {Image: "e"}, "svc-f": {Image: "f"},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "too many")
}

func TestValidate_ServicesInvalidName(t *testing.T) {
	af := validAgentfile()
	af.Services = map[string]agentfile.Service{
		"My-Service": {Image: "redis:7"},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "DNS")
}

func TestParse_ValidServices(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_services.yaml"))
	require.NoError(t, err)
	require.Len(t, af.Services, 2)
	assert.Equal(t, "redis:7-alpine", af.Services["redis"].Image)
	assert.Equal(t, "postgres:16", af.Services["db"].Image)
	assert.Equal(t, 5432, af.Services["db"].Port)
	assert.Equal(t, []string{"/var/lib/postgresql/data"}, af.Services["db"].Volumes)
	assert.Equal(t, []string{"POSTGRES_PASSWORD"}, af.Services["db"].Env)
}

func TestLoad_InvalidServicesReserved(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_services_reserved.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "reserved")
}

func TestLoad_InvalidServicesNoImage(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_services_no_image.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "image")
}

func TestLoad_InvalidServicesPortConflict(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_services_port_conflict.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "port")
}

func TestLoad_InvalidServicesTooMany(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_services_too_many.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "too many")
}

func TestLoad_BackwardCompatNoServices(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Services)
}

// --- Security parse tests ---

func TestParse_ValidSecurity(t *testing.T) {
	r := openFixture(t, "valid_security.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	require.NotNil(t, af.Security)
	assert.True(t, af.Security.ReadOnly)
	assert.True(t, af.Security.NoNewPrivileges)
	assert.Equal(t, []string{"ALL"}, af.Security.DropCapabilities)
}

func TestParse_ValidSecurityPartial(t *testing.T) {
	r := openFixture(t, "valid_security_partial.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	require.NotNil(t, af.Security)
	assert.True(t, af.Security.ReadOnly)
	assert.False(t, af.Security.NoNewPrivileges)
	assert.Nil(t, af.Security.DropCapabilities)
}

func TestLoad_BackwardCompatNoSecurity(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Security)
}

// --- GPU parse tests ---

func TestParse_ValidGPU(t *testing.T) {
	r := openFixture(t, "valid_gpu.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	assert.True(t, af.GPU)
}

func TestLoad_BackwardCompatNoGPU(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.False(t, af.GPU)
}

// --- Security validation tests ---

func TestValidate_SecurityValid(t *testing.T) {
	af := validAgentfile()
	af.Security = &agentfile.SecurityContext{
		ReadOnly:         true,
		NoNewPrivileges:  true,
		DropCapabilities: []string{"ALL"},
	}
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_SecurityEmptyStruct(t *testing.T) {
	af := validAgentfile()
	af.Security = &agentfile.SecurityContext{}
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_SecurityInvalidCapability(t *testing.T) {
	af := validAgentfile()
	af.Security = &agentfile.SecurityContext{
		DropCapabilities: []string{"invalid_lower"},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "not a valid Linux capability")
}

func TestValidate_SecurityEmptyCapability(t *testing.T) {
	af := validAgentfile()
	af.Security = &agentfile.SecurityContext{
		DropCapabilities: []string{""},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "empty entry")
}

func TestValidate_SecurityDuplicateCapability(t *testing.T) {
	af := validAgentfile()
	af.Security = &agentfile.SecurityContext{
		DropCapabilities: []string{"ALL", "ALL"},
	}
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "duplicate")
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

// --- PackageManager tests ---

func TestParse_ValidPackageManager(t *testing.T) {
	cases := []struct {
		input    string
		expected agentfile.PackageManager
	}{
		{"pip", agentfile.PackageManagerPip},
		{"poetry", agentfile.PackageManagerPoetry},
		{"uv", agentfile.PackageManagerUV},
		{"pipenv", agentfile.PackageManagerPipenv},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			yaml := fmt.Sprintf("version: 1\nname: ab\nframework: generic\nport: 8000\nhealth_path: /h\npackage_manager: %s\ndockerfile: auto\n", tc.input)
			af, err := agentfile.Parse(strings.NewReader(yaml))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, af.PackageManager)
		})
	}
}

func TestParse_InvalidPackageManager(t *testing.T) {
	r := openFixture(t, "invalid_package_manager.yaml")
	_, err := agentfile.Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package_manager")
}

func TestParse_ValidPackageManagerFromFixture(t *testing.T) {
	r := openFixture(t, "valid_package_manager.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	assert.Equal(t, agentfile.PackageManagerPoetry, af.PackageManager)
}

func TestLoad_BackwardCompat_NoPackageManager(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Equal(t, agentfile.PackageManager(""), af.PackageManager)
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

// --- Healthcheck parse tests ---

func TestParse_ValidServiceHealthcheck(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_service_healthcheck.yaml"))
	require.NoError(t, err)
	require.NotNil(t, af.Services["db"].Healthcheck)
	assert.Equal(t, []string{"CMD-SHELL", "pg_isready -U myuser || exit 1"}, af.Services["db"].Healthcheck.Test)
	assert.Equal(t, "10s", af.Services["db"].Healthcheck.Interval)
	assert.Equal(t, 3, af.Services["db"].Healthcheck.Retries)
}

func TestLoad_BackwardCompatNoHealthcheck(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_services.yaml"))
	require.NoError(t, err)
	// Existing services without healthcheck should still work
	assert.Nil(t, af.Services["redis"].Healthcheck)
}

// --- Build parse and validate tests ---

func TestParse_ValidBuild(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_build.yaml"))
	require.NoError(t, err)
	require.NotNil(t, af.Build)
	assert.Len(t, af.Build.SetupCommands, 2)
	assert.Len(t, af.Build.CacheDirs, 2)
}

func TestLoad_BackwardCompatNoBuild(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Build)
}

func TestLoad_InvalidBuildEmptyCommand(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_build_empty_command.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "empty command")
}

func TestLoad_InvalidBuildNotAuto(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_build_not_auto.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "dockerfile: auto")
}

func TestLoad_InvalidBuildCacheNotAbsolute(t *testing.T) {
	_, err := agentfile.Load(fixturePath("invalid_build_cache_not_absolute.yaml"))
	require.Error(t, err)
	requireUserError(t, err, output.CodeInvalidAgentfile, "absolute path")
}

func TestValidate_BuildValid(t *testing.T) {
	af := validAgentfile()
	af.Build = &agentfile.BuildConfig{
		SetupCommands: []string{"echo hello"},
		CacheDirs:     []string{"/root/.cache"},
	}
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_BuildNil(t *testing.T) {
	af := validAgentfile()
	af.Build = nil
	assert.NoError(t, agentfile.Validate(af))
}

// --- HostPort parse and validate tests ---

func TestParse_ValidHostPort(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_host_port.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 18000, af.HostPort)
	assert.Equal(t, 16379, af.Services["redis"].HostPort)
}

func TestLoad_BackwardCompatNoHostPort(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 0, af.HostPort)
}

func TestValidate_HostPortValid(t *testing.T) {
	af := validAgentfile()
	af.HostPort = 18000
	assert.NoError(t, agentfile.Validate(af))
}

func TestValidate_HostPortOutOfRange(t *testing.T) {
	af := validAgentfile()
	af.HostPort = 70000
	err := agentfile.Validate(af)
	requireUserError(t, err, output.CodeInvalidAgentfile, "host_port")
}

// --- Resource parse tests ---

func TestParse_ValidResources(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_resources.yaml"))
	require.NoError(t, err)
	require.NotNil(t, af.Services["db"].Resources)
	assert.Equal(t, "1g", af.Services["db"].Resources.MemLimit)
	assert.Equal(t, "1.0", af.Services["db"].Resources.CPUs)
}

func TestLoad_BackwardCompatNoResources(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_services.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Services["redis"].Resources)
}

// --- Tmpfs parse tests ---

func TestParse_ValidTmpfs(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_tmpfs.yaml"))
	require.NoError(t, err)
	require.NotNil(t, af.Security)
	require.Len(t, af.Security.Tmpfs, 2)
	assert.Equal(t, "/tmp", af.Security.Tmpfs[0].Path)
	assert.Equal(t, "200M", af.Security.Tmpfs[0].Size)
}

func TestLoad_BackwardCompatNoTmpfs(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_security.yaml"))
	require.NoError(t, err)
	assert.Empty(t, af.Security.Tmpfs)
}

// --- Observability tests ---

func TestParse_ValidObservability(t *testing.T) {
	r := openFixture(t, "valid_observability.yaml")
	af, err := agentfile.Parse(r)
	require.NoError(t, err)
	require.NotNil(t, af.Observability)
	assert.Equal(t, 2, af.Observability.Level)
	assert.Equal(t, 9101, af.Observability.MetricsPort)
}

func TestLoad_BackwardCompatNoObservability(t *testing.T) {
	af, err := agentfile.Load(fixturePath("valid_minimal.yaml"))
	require.NoError(t, err)
	assert.Nil(t, af.Observability)
}

func TestValidate_ObservabilityInvalidLevel(t *testing.T) {
	af := validAgentfile()
	af.Observability = &agentfile.ObservabilityConfig{Level: 5}
	err := agentfile.Validate(af)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "observability.level")
}

func TestValidate_ObservabilityInvalidPort(t *testing.T) {
	af := validAgentfile()
	af.Observability = &agentfile.ObservabilityConfig{Level: 2, MetricsPort: 99999}
	err := agentfile.Validate(af)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "observability.metrics_port")
}

func TestValidate_ObservabilityLevel2GenericFramework(t *testing.T) {
	// Level 2 must work with generic framework (no framework constraint).
	af := validAgentfile()
	af.Framework = agentfile.FrameworkGeneric
	af.Observability = &agentfile.ObservabilityConfig{Level: 2}
	err := agentfile.Validate(af)
	require.NoError(t, err)
}
