package setup

import (
	"testing"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/stretchr/testify/assert"
)

func TestDetectPackageManager_UV(t *testing.T) {
	pm := detectPackageManager(fixture("uv_project"))
	assert.Equal(t, agentfile.PackageManagerUV, pm)
}

func TestDetectPackageManager_Poetry(t *testing.T) {
	pm := detectPackageManager(fixture("poetry_project"))
	assert.Equal(t, agentfile.PackageManagerPoetry, pm)
}

func TestDetectPackageManager_Pipenv(t *testing.T) {
	pm := detectPackageManager(fixture("pipenv_project"))
	assert.Equal(t, agentfile.PackageManagerPipenv, pm)
}

func TestDetectPackageManager_Pip(t *testing.T) {
	pm := detectPackageManager(fixture("fastapi_project"))
	assert.Equal(t, agentfile.PackageManagerPip, pm)
}

func TestDetectPackageManager_PriorityUVOverPoetry(t *testing.T) {
	pm := detectPackageManager(fixture("uv_poetry_project"))
	assert.Equal(t, agentfile.PackageManagerUV, pm)
}

func TestDetectPackageManager_EmptyProject(t *testing.T) {
	pm := detectPackageManager(fixture("empty_project"))
	assert.Equal(t, agentfile.PackageManagerPip, pm)
}

func TestDetectPackageManager_Nonexistent(t *testing.T) {
	pm := detectPackageManager("/nonexistent/dir")
	assert.Equal(t, agentfile.PackageManagerPip, pm)
}
