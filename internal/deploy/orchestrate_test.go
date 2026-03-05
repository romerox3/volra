package deploy

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrate_Success(t *testing.T) {
	dir := t.TempDir()
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s/.volra/docker-compose.yml -p my-agent up -d --build", dir): {
				Output: "Creating my-agent-agent ... done\n", Err: nil,
			},
		},
	}

	err := Orchestrate(context.Background(), mock, "my-agent", dir)
	require.NoError(t, err)
	require.Len(t, mock.Calls, 1)
	assert.Equal(t, "compose", mock.Calls[0][0])
}

func TestOrchestrate_DockerNotRunning(t *testing.T) {
	dir := t.TempDir()
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s/.volra/docker-compose.yml -p test up -d --build", dir): {
				Output: "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?",
				Err:    errors.New("exit status 1"),
			},
		},
	}

	err := Orchestrate(context.Background(), mock, "test", dir)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeDeployDockerNotRunning, ue.Code)
	assert.Contains(t, ue.What, "Docker is not running")
}

func TestOrchestrate_BuildFailed(t *testing.T) {
	dir := t.TempDir()
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s/.volra/docker-compose.yml -p test up -d --build", dir): {
				Output: "Step 4/6 : RUN pip install -r requirements.txt\nERROR: Could not find a version that satisfies the requirement nonexistent-pkg\nbuild failed",
				Err:    errors.New("exit status 1"),
			},
		},
	}

	err := Orchestrate(context.Background(), mock, "test", dir)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeBuildFailed, ue.Code)
	assert.Contains(t, ue.What, "build failed")
}

func TestOrchestrate_UnknownError(t *testing.T) {
	dir := t.TempDir()
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s/.volra/docker-compose.yml -p test up -d --build", dir): {
				Output: "some unknown error output",
				Err:    errors.New("exit status 1"),
			},
		},
	}

	err := Orchestrate(context.Background(), mock, "test", dir)
	require.Error(t, err)

	// Should NOT be a UserError — falls through to generic wrap
	var ue *output.UserError
	assert.False(t, errors.As(err, &ue))
	assert.Contains(t, err.Error(), "docker compose up failed")
}

func TestOrchestrate_CommandArgs(t *testing.T) {
	dir := t.TempDir()
	mock := &testutil.MockDockerRunner{
		Responses: map[string]testutil.MockResponse{
			fmt.Sprintf("compose -f %s/.volra/docker-compose.yml -p cool-agent up -d --build", dir): {
				Output: "", Err: nil,
			},
		},
	}

	err := Orchestrate(context.Background(), mock, "cool-agent", dir)
	require.NoError(t, err)

	require.Len(t, mock.Calls, 1)
	args := mock.Calls[0]
	assert.Equal(t, []string{
		"compose", "-f", fmt.Sprintf("%s/.volra/docker-compose.yml", dir),
		"-p", "cool-agent", "up", "-d", "--build",
	}, args)
}

func TestExtractExcerpt_Short(t *testing.T) {
	got := extractExcerpt("line1\nline2", 5)
	assert.Equal(t, "line1\nline2", got)
}

func TestExtractExcerpt_Long(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	got := extractExcerpt(input, 3)
	assert.Equal(t, "line5\nline6\nline7", got)
}
