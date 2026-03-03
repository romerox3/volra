package deploy

const (
	// JobHealth is the Prometheus job name for health endpoint scraping.
	JobHealth = "agent-health"
	// JobMetrics is the Prometheus job name for metrics scraping.
	JobMetrics = "agent-metrics"
	// DatasourceName is the Grafana datasource name.
	DatasourceName = "Prometheus"
	// NetworkName is the Docker network name.
	NetworkName = "volra"
	// OutputDir is the generated artifacts directory name.
	OutputDir = ".volra"
)
