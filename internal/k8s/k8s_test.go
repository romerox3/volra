package k8s

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basicContext() *ManifestContext {
	return &ManifestContext{
		Name:       "my-agent",
		Image:      "my-agent:latest",
		Port:       8000,
		HealthPath: "/health",
		Replicas:   1,
	}
}

func TestRenderDeployment_Basic(t *testing.T) {
	out, err := RenderDeployment(basicContext())
	require.NoError(t, err)

	assert.Contains(t, out, "kind: Deployment")
	assert.Contains(t, out, "name: my-agent")
	assert.Contains(t, out, "image: my-agent:latest")
	assert.Contains(t, out, "containerPort: 8000")
	assert.Contains(t, out, "path: /health")
	assert.Contains(t, out, "livenessProbe")
	assert.Contains(t, out, "readinessProbe")
}

func TestRenderDeployment_WithEnv(t *testing.T) {
	ctx := basicContext()
	ctx.Env = []EnvVar{{Name: "API_KEY", Value: "secret"}}

	out, err := RenderDeployment(ctx)
	require.NoError(t, err)

	assert.Contains(t, out, "configMapRef")
	assert.Contains(t, out, "my-agent-config")
}

func TestRenderDeployment_WithVolumes(t *testing.T) {
	ctx := basicContext()
	ctx.Volumes = []VolumeMount{{Name: "data", MountPath: "/data", Size: "5Gi"}}

	out, err := RenderDeployment(ctx)
	require.NoError(t, err)

	assert.Contains(t, out, "volumeMounts")
	assert.Contains(t, out, "mountPath: /data")
	assert.Contains(t, out, "persistentVolumeClaim")
}

func TestRenderDeployment_WithNamespace(t *testing.T) {
	ctx := basicContext()
	ctx.Namespace = "team-a"

	out, err := RenderDeployment(ctx)
	require.NoError(t, err)

	assert.Contains(t, out, "namespace: team-a")
}

func TestRenderService(t *testing.T) {
	out, err := RenderService(basicContext())
	require.NoError(t, err)

	assert.Contains(t, out, "kind: Service")
	assert.Contains(t, out, "type: ClusterIP")
	assert.Contains(t, out, "port: 8000")
}

func TestRenderConfigMap_WithEnv(t *testing.T) {
	ctx := basicContext()
	ctx.Env = []EnvVar{
		{Name: "DB_HOST", Value: "localhost"},
		{Name: "DB_PORT", Value: "5432"},
	}

	out, err := RenderConfigMap(ctx)
	require.NoError(t, err)

	assert.Contains(t, out, "kind: ConfigMap")
	assert.Contains(t, out, "DB_HOST")
	assert.Contains(t, out, "DB_PORT")
}

func TestRenderConfigMap_NoEnvReturnsEmpty(t *testing.T) {
	out, err := RenderConfigMap(basicContext())
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRenderPVC(t *testing.T) {
	ctx := basicContext()
	ctx.Volumes = []VolumeMount{
		{Name: "data", MountPath: "/data", Size: "10Gi"},
	}

	out, err := RenderPVC(ctx)
	require.NoError(t, err)

	assert.Contains(t, out, "kind: PersistentVolumeClaim")
	assert.Contains(t, out, "storage: 10Gi")
}

func TestRenderPVC_NoVolumesReturnsEmpty(t *testing.T) {
	out, err := RenderPVC(basicContext())
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRenderPVC_DefaultSize(t *testing.T) {
	ctx := basicContext()
	ctx.Volumes = []VolumeMount{
		{Name: "data", MountPath: "/data"},
	}

	out, err := RenderPVC(ctx)
	require.NoError(t, err)
	assert.Contains(t, out, "storage: 1Gi")
}

func TestRenderServiceMonitor(t *testing.T) {
	out, err := RenderServiceMonitor(basicContext())
	require.NoError(t, err)

	assert.Contains(t, out, "kind: ServiceMonitor")
	assert.Contains(t, out, "monitoring.coreos.com/v1")
	assert.Contains(t, out, "path: /metrics")
}

func TestGenerateAll_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := basicContext()
	ctx.Env = []EnvVar{{Name: "KEY", Value: "val"}}
	ctx.Volumes = []VolumeMount{{Name: "data", MountPath: "/data"}}

	err := GenerateAll(ctx, dir)
	require.NoError(t, err)

	k8sDir := filepath.Join(dir, ".volra", "k8s")
	assert.FileExists(t, filepath.Join(k8sDir, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "service.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "configmap.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "pvc.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "servicemonitor.yaml"))
}

func TestGenerateAll_NoOptionalFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := basicContext()

	err := GenerateAll(ctx, dir)
	require.NoError(t, err)

	k8sDir := filepath.Join(dir, ".volra", "k8s")
	assert.FileExists(t, filepath.Join(k8sDir, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "service.yaml"))
	assert.FileExists(t, filepath.Join(k8sDir, "servicemonitor.yaml"))

	// Optional files should NOT exist.
	_, err = os.Stat(filepath.Join(k8sDir, "configmap.yaml"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(k8sDir, "pvc.yaml"))
	assert.True(t, os.IsNotExist(err))
}
