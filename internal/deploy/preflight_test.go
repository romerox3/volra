package deploy

import (
	"context"
	"fmt"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckGPUAvailable_Success(t *testing.T) {
	dr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info --format {{.Runtimes}}": {Output: "map[io.containerd.runc.v2:{} nvidia:{}]"},
		},
	}
	err := CheckGPUAvailable(context.Background(), dr)
	assert.NoError(t, err)
}

func TestCheckGPUAvailable_NoNvidia(t *testing.T) {
	dr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info --format {{.Runtimes}}": {Output: "map[io.containerd.runc.v2:{}]"},
		},
	}
	err := CheckGPUAvailable(context.Background(), dr)
	require.Error(t, err)
	ue, ok := err.(*output.UserError)
	require.True(t, ok)
	assert.Equal(t, output.CodeGPUNotAvailable, ue.Code)
}

func TestCheckGPUAvailable_DockerError(t *testing.T) {
	dr := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			"info --format {{.Runtimes}}": {Err: fmt.Errorf("cannot connect")},
		},
	}
	err := CheckGPUAvailable(context.Background(), dr)
	require.Error(t, err)
	ue, ok := err.(*output.UserError)
	require.True(t, ok)
	assert.Equal(t, output.CodeGPUCheckFailed, ue.Code)
}
