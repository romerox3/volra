package doctor

import (
	"context"
	"errors"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSystemInfo provides controllable system check responses.
type mockSystemInfo struct {
	pythonVersion string
	pythonErr     error
	diskGB        float64
	diskErr       error
	freePorts     map[int]bool // true = free
}

func (m *mockSystemInfo) PythonVersion(_ context.Context) (string, error) {
	return m.pythonVersion, m.pythonErr
}

func (m *mockSystemInfo) AvailableDiskGB() (float64, error) {
	return m.diskGB, m.diskErr
}

func (m *mockSystemInfo) IsPortFree(port int) bool {
	if m.freePorts == nil {
		return true
	}
	free, ok := m.freePorts[port]
	if !ok {
		return true
	}
	return free
}

func allHealthyDocker() map[string]testutil.MockResponse {
	return map[string]testutil.MockResponse{
		"--version":              {Output: "Docker version 27.5.1, build abc123", Err: nil},
		"info":                   {Output: "Client: Docker Engine", Err: nil},
		"compose version":        {Output: "Docker Compose version v2.32.4", Err: nil},
		"compose version --short": {Output: "2.32.4", Err: nil},
	}
}

func healthySystem() *mockSystemInfo {
	return &mockSystemInfo{
		pythonVersion: "Python 3.11.5",
		diskGB:        45.0,
		freePorts:     map[int]bool{9090: true, 3001: true},
	}
}

func TestRun_AllChecksPassed(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}

	err := Run(context.Background(), "v0.1.0", mp, mr, healthySystem())

	require.NoError(t, err)
	assert.Len(t, mp.ErrorCalls, 0)
	assert.NotEmpty(t, mp.ResultCalls)
	assert.Contains(t, mp.ResultCalls[0], "All checks passed")
}

func TestRun_VersionReported(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}

	_ = Run(context.Background(), "v0.1.0", mp, mr, healthySystem())

	found := false
	for _, msg := range mp.ProgressCalls {
		if assert.ObjectsAreEqual("  Version: v0.1.0", msg) {
			found = true
		}
	}
	assert.True(t, found, "version should be reported in progress output")
}

func TestRun_DockerNotInstalled(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"--version":       {Output: "", Err: errors.New("command not found")},
			"info":            {Output: "", Err: errors.New("command not found")},
			"compose version": {Output: "", Err: errors.New("command not found")},
		},
	}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.Error(t, err)
	require.NotEmpty(t, mp.ErrorCalls)

	var ue *output.UserError
	require.True(t, errors.As(mp.ErrorCalls[0], &ue))
	assert.Equal(t, output.CodeDockerNotInstalled, ue.Code)
}

func TestRun_DockerNotRunning(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"--version":       {Output: "Docker version 27.5.1", Err: nil},
			"info":            {Output: "", Err: errors.New("Cannot connect to Docker daemon")},
			"compose version": {Output: "", Err: errors.New("Cannot connect to Docker daemon")},
		},
	}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.Error(t, err)
	foundE102 := false
	for _, e := range mp.ErrorCalls {
		var ue *output.UserError
		if errors.As(e, &ue) && ue.Code == output.CodeDockerNotRunning {
			foundE102 = true
		}
	}
	assert.True(t, foundE102, "should report E102 for Docker not running")
}

func TestRun_ComposeNotAvailable(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"--version":       {Output: "Docker version 27.5.1", Err: nil},
			"info":            {Output: "Client: Docker Engine", Err: nil},
			"compose version": {Output: "", Err: errors.New("not a docker command")},
		},
	}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.Error(t, err)
	foundE103 := false
	for _, e := range mp.ErrorCalls {
		var ue *output.UserError
		if errors.As(e, &ue) && ue.Code == output.CodeComposeNotAvailable {
			foundE103 = true
		}
	}
	assert.True(t, foundE103, "should report E103 for Compose not available")
}

func TestRun_PythonNotInstalled(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}
	sys := healthySystem()
	sys.pythonErr = errors.New("command not found")
	sys.pythonVersion = ""

	err := Run(context.Background(), "", mp, mr, sys)

	require.Error(t, err)
	foundE105 := false
	for _, e := range mp.ErrorCalls {
		var ue *output.UserError
		if errors.As(e, &ue) && ue.Code == output.CodePythonNotFound {
			foundE105 = true
		}
	}
	assert.True(t, foundE105, "should report E105 for Python not found")
}

func TestRun_PythonTooOld(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}
	sys := healthySystem()
	sys.pythonVersion = "Python 3.8.10"

	err := Run(context.Background(), "", mp, mr, sys)

	require.Error(t, err)
	foundE105 := false
	for _, e := range mp.ErrorCalls {
		var ue *output.UserError
		if errors.As(e, &ue) && ue.Code == output.CodePythonNotFound {
			foundE105 = true
		}
	}
	assert.True(t, foundE105, "should report E105 for Python < 3.10")
}

func TestRun_InsufficientDiskSpace(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}
	sys := healthySystem()
	sys.diskGB = 0.5

	err := Run(context.Background(), "", mp, mr, sys)

	require.Error(t, err)
	foundE106 := false
	for _, e := range mp.ErrorCalls {
		var ue *output.UserError
		if errors.As(e, &ue) && ue.Code == output.CodeInsufficientDisk {
			foundE106 = true
		}
	}
	assert.True(t, foundE106, "should report E106 for insufficient disk")
}

func TestRun_PortInUseWarns(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}
	sys := healthySystem()
	sys.freePorts[9090] = false

	err := Run(context.Background(), "", mp, mr, sys)

	require.NoError(t, err, "port warnings should not cause failure")
	require.NotEmpty(t, mp.WarnCalls)
	assert.Contains(t, mp.WarnCalls[0].What, "9090")
}

func TestRun_MultiplePortWarnings(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}
	sys := healthySystem()
	sys.freePorts[9090] = false
	sys.freePorts[3001] = false

	err := Run(context.Background(), "", mp, mr, sys)

	require.NoError(t, err, "port warnings should not cause failure")
	assert.Len(t, mp.WarnCalls, 2)
}

func TestIsPythonVersionOK(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Python 3.10.5", true},
		{"Python 3.11.0", true},
		{"Python 3.12.1", true},
		{"Python 3.9.7", false},
		{"Python 3.8.10", false},
		{"Python 2.7.18", false},
		{"Python 4.0.0", true},
		{"garbage", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isPythonVersionOK(tt.input))
		})
	}
}

func TestCheckComposeWatchVersion_Supported(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{Responses: allHealthyDocker()}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.NoError(t, err)
	// Should have a progress message about watch being supported
	foundWatch := false
	for _, msg := range mp.ProgressCalls {
		if assert.ObjectsAreEqual("  ✓ Docker Compose 2.32.4 (watch supported)", msg) {
			foundWatch = true
		}
	}
	assert.True(t, foundWatch, "should report compose watch supported")
}

func TestCheckComposeWatchVersion_TooOld(t *testing.T) {
	mp := &testutil.MockPresenter{}
	responses := allHealthyDocker()
	responses["compose version --short"] = testutil.MockResponse{Output: "2.20.3", Err: nil}
	mr := &testutil.MockDockerRunner{Responses: responses}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.NoError(t, err, "old compose watch should warn, not fail")
	foundWarn := false
	for _, w := range mp.WarnCalls {
		if w.What == "Docker Compose 2.20.3 does not support watch (requires >= 2.22.0)" {
			foundWarn = true
		}
	}
	assert.True(t, foundWarn, "should warn about old compose version for watch")
}

func TestCheckComposeWatchVersion_ExactMinimum(t *testing.T) {
	mp := &testutil.MockPresenter{}
	responses := allHealthyDocker()
	responses["compose version --short"] = testutil.MockResponse{Output: "2.22.0", Err: nil}
	mr := &testutil.MockDockerRunner{Responses: responses}

	err := Run(context.Background(), "", mp, mr, healthySystem())

	require.NoError(t, err)
	// Should pass, not warn
	for _, w := range mp.WarnCalls {
		assert.NotContains(t, w.What, "watch", "exact minimum 2.22.0 should pass, not warn")
	}
}

func TestIsComposeVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.32.4", true},
		{"2.22.0", true},
		{"2.22.1", true},
		{"2.23.0", true},
		{"3.0.0", true},
		{"2.21.9", false},
		{"2.20.3", false},
		{"1.29.0", false},
		{"v2.29.1", true},
		{"2.22.0-beta.1", true},
		{"garbage", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsComposeVersionAtLeast(tt.version, 2, 22, 0))
		})
	}
}

func TestRun_AllChecksRunEvenOnFailure(t *testing.T) {
	mp := &testutil.MockPresenter{}
	mr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"--version":               {Output: "", Err: errors.New("not found")},
			"info":                    {Output: "", Err: errors.New("not found")},
			"compose version":         {Output: "", Err: errors.New("not found")},
			"compose version --short": {Output: "", Err: errors.New("not found")},
		},
	}
	sys := healthySystem()
	sys.pythonErr = errors.New("not found")
	sys.diskGB = 0.1

	_ = Run(context.Background(), "", mp, mr, sys)

	// Should have errors for Docker, Docker running, Compose, Python, and Disk
	assert.GreaterOrEqual(t, len(mp.ErrorCalls), 5, "all 5 failing checks should be reported")
}
