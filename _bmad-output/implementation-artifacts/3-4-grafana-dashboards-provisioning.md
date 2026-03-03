# Story 3.4: Grafana Dashboards & Provisioning

Status: done

## Story

**As a** developer deploying with Volra,
**I want** pre-built Grafana dashboards showing agent health, uptime, and latency,
**So that** I get instant observability without manual dashboard configuration.

## Acceptance Criteria (BDD)

```gherkin
Given the deploy command generates Grafana assets
When I inspect the output
Then the following files are generated in .volra/:
  grafana/provisioning/datasources/datasource.yml
  grafana/provisioning/dashboards/dashboards.yml
  grafana/dashboards/overview.json
  grafana/dashboards/detail.json

Given the Overview dashboard (overview.json)
When loaded in Grafana
Then it shows: agent status badge (healthy/unhealthy), uptime, current probe latency
  And all metrics are clearly labeled as probe-based

Given the Detail dashboard (detail.json)
When loaded in Grafana
Then it shows: probe latency over time, health status timeline, uptime percentage for 24h and 7d
  And all metrics are clearly labeled as probe-based

Given datasource.yml
When Grafana loads provisioning
Then it auto-configures Prometheus at http://prometheus:9090 as default datasource

Given dashboards.yml
When Grafana loads provisioning
Then it discovers dashboard JSON files from the provisioned path

Given the Overview dashboard
When Grafana starts
Then it is set as the default home dashboard (via GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH in compose)
```

## Tasks / Subtasks

- [x] **Task 1: Create static/datasource.yml** — Prometheus datasource (http://prometheus:9090, isDefault: true)
- [x] **Task 2: Create static/dashboards.yml** — Dashboard provider pointing to /var/lib/grafana/dashboards
- [x] **Task 3: Create static/overview.json** — 3 panels: status badge, uptime 24h, probe latency
- [x] **Task 4: Create static/detail.json** — 4 panels: latency over time, health timeline, uptime 24h, uptime 7d
- [x] **Task 5: Create grafana.go** — CopyGrafanaAssets() with asset table mapping src→dst
- [x] **Task 6: Create grafana_test.go** — 7 tests covering files, content, structure, probe labeling
- [x] **Task 7: Update consistency_test.go** — Added 4 new static files (total: 5 static + 3 templates)
- [x] **Task 8: Verify lint + test pass** — 178 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **All 4 files are static** — no Go templating. Embedded via `staticFS` and copied to output.
2. **grafanaAssets table** drives CopyGrafanaAssets() — maps `static/X` → `grafana/Y` paths.
3. **Output mirrors Grafana provisioning layout**: `grafana/provisioning/datasources/`, `grafana/provisioning/dashboards/`, `grafana/dashboards/`.
4. **Overview dashboard** has 3 panels: Agent Status (stat with UP/DOWN mapping), Uptime 24h (percentunit), Probe Latency (seconds).
5. **Detail dashboard** has 4 panels: Probe Latency Over Time (timeseries), Health Status Timeline (state-timeline), Uptime 24h, Uptime 7d.
6. **FR37 compliance**: All panel titles include "Probe" to clearly label metrics as probe-based.
7. **consistency_test.go** now verifies 3 templates + 5 static files.

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/static/datasource.yml` | Created | Prometheus datasource provisioning |
| `internal/deploy/static/dashboards.yml` | Created | Dashboard provider config |
| `internal/deploy/static/overview.json` | Created | Overview dashboard (3 panels) |
| `internal/deploy/static/detail.json` | Created | Detail dashboard (4 panels) |
| `internal/deploy/grafana.go` | Created | CopyGrafanaAssets function |
| `internal/deploy/grafana_test.go` | Created | 7 tests |
| `internal/deploy/consistency_test.go` | Modified | Added 4 new static file checks |
