// Package k8s generates Kubernetes manifests from Agentfile configuration.
package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ManifestContext holds the data needed for K8s manifest generation.
type ManifestContext struct {
	Name        string
	Image       string
	Port        int
	HealthPath  string
	Env         []EnvVar
	Volumes     []VolumeMount
	Replicas    int
	Namespace   string
}

// EnvVar represents a key-value environment variable.
type EnvVar struct {
	Name  string
	Value string
}

// VolumeMount represents a persistent volume claim mount.
type VolumeMount struct {
	Name      string
	MountPath string
	Size      string
}

const deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
{{- if .Namespace}}
  namespace: {{.Namespace}}
{{- end}}
  labels:
    app: {{.Name}}
    managed-by: volra
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      app: {{.Name}}
  template:
    metadata:
      labels:
        app: {{.Name}}
    spec:
      containers:
        - name: {{.Name}}
          image: {{.Image}}
          ports:
            - containerPort: {{.Port}}
          livenessProbe:
            httpGet:
              path: {{.HealthPath}}
              port: {{.Port}}
            initialDelaySeconds: 10
            periodSeconds: 15
            timeoutSeconds: 5
          readinessProbe:
            httpGet:
              path: {{.HealthPath}}
              port: {{.Port}}
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 3
{{- if .Env}}
          envFrom:
            - configMapRef:
                name: {{.Name}}-config
{{- end}}
{{- if .Volumes}}
          volumeMounts:
{{- range .Volumes}}
            - name: {{.Name}}
              mountPath: {{.MountPath}}
{{- end}}
{{- end}}
{{- if .Volumes}}
      volumes:
{{- range .Volumes}}
        - name: {{.Name}}
          persistentVolumeClaim:
            claimName: {{.Name}}
{{- end}}
{{- end}}
`

const serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
{{- if .Namespace}}
  namespace: {{.Namespace}}
{{- end}}
  labels:
    app: {{.Name}}
    managed-by: volra
spec:
  selector:
    app: {{.Name}}
  ports:
    - port: {{.Port}}
      targetPort: {{.Port}}
      protocol: TCP
  type: ClusterIP
`

const configMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}-config
{{- if .Namespace}}
  namespace: {{.Namespace}}
{{- end}}
  labels:
    app: {{.Name}}
    managed-by: volra
data:
{{- range .Env}}
  {{.Name}}: "{{.Value}}"
{{- end}}
`

const pvcTemplate = `{{- range .Volumes}}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{.Name}}
  labels:
    managed-by: volra
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: {{if .Size}}{{.Size}}{{else}}1Gi{{end}}
---
{{- end}}
`

const serviceMonitorTemplate = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{.Name}}
{{- if .Namespace}}
  namespace: {{.Namespace}}
{{- end}}
  labels:
    app: {{.Name}}
    managed-by: volra
spec:
  selector:
    matchLabels:
      app: {{.Name}}
  endpoints:
    - port: "{{.Port}}"
      path: /metrics
      interval: 15s
`

// RenderDeployment generates a Deployment manifest.
func RenderDeployment(ctx *ManifestContext) (string, error) {
	return render("deployment", deploymentTemplate, ctx)
}

// RenderService generates a Service manifest.
func RenderService(ctx *ManifestContext) (string, error) {
	return render("service", serviceTemplate, ctx)
}

// RenderConfigMap generates a ConfigMap manifest (only if env vars exist).
func RenderConfigMap(ctx *ManifestContext) (string, error) {
	if len(ctx.Env) == 0 {
		return "", nil
	}
	return render("configmap", configMapTemplate, ctx)
}

// RenderPVC generates PersistentVolumeClaim manifests (only if volumes exist).
func RenderPVC(ctx *ManifestContext) (string, error) {
	if len(ctx.Volumes) == 0 {
		return "", nil
	}
	return render("pvc", pvcTemplate, ctx)
}

// RenderServiceMonitor generates a ServiceMonitor CRD manifest.
func RenderServiceMonitor(ctx *ManifestContext) (string, error) {
	return render("servicemonitor", serviceMonitorTemplate, ctx)
}

// GenerateAll writes all K8s manifests to the output directory.
func GenerateAll(ctx *ManifestContext, dir string) error {
	outputDir := filepath.Join(dir, ".volra", "k8s")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating k8s output directory: %w", err)
	}

	manifests := map[string]func(*ManifestContext) (string, error){
		"deployment.yaml":     RenderDeployment,
		"service.yaml":        RenderService,
		"servicemonitor.yaml": RenderServiceMonitor,
	}

	for filename, renderFn := range manifests {
		content, err := renderFn(ctx)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", filename, err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, filename), []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}

	// Optional manifests.
	if cm, err := RenderConfigMap(ctx); err != nil {
		return fmt.Errorf("rendering configmap: %w", err)
	} else if cm != "" {
		if err := os.WriteFile(filepath.Join(outputDir, "configmap.yaml"), []byte(cm), 0o644); err != nil {
			return fmt.Errorf("writing configmap.yaml: %w", err)
		}
	}

	if pvc, err := RenderPVC(ctx); err != nil {
		return fmt.Errorf("rendering pvc: %w", err)
	} else if pvc != "" {
		if err := os.WriteFile(filepath.Join(outputDir, "pvc.yaml"), []byte(pvc), 0o644); err != nil {
			return fmt.Errorf("writing pvc.yaml: %w", err)
		}
	}

	return nil
}

func render(name, tmplStr string, ctx *ManifestContext) (string, error) {
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing %s template: %w", name, err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("rendering %s: %w", name, err)
	}
	return buf.String(), nil
}
