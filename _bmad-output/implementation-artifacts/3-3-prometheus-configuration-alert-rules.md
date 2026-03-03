# Story 3.3: Prometheus Configuration & Alert Rules

Status: done

## Story

**As a** Volra developer,
**I want** the deploy command to generate Prometheus config that scrapes the agent's health endpoint,
**So that** probe metrics (up, latency) are collected automatically.

## Acceptance Criteria (BDD)

```gherkin
Given a valid Agentfile with health_path: /health
When the Prometheus config generator runs
Then prometheus.yml is generated with a scrape job targeting http://agent:{port}{health_path}
  And scrape_interval is 15s (NFR5)
  And scrape_timeout is 10s (NFR5)
  And rule_files references /etc/prometheus/alert_rules.yml

Given the generated alert_rules.yml (static file)
When I inspect it
Then it contains an alert rule for agent-down detection (FR20)
  And the alert fires when `up == 0` for the health job for > 1 minute (NFR11)

Given the agent exposes /metrics in Prometheus format
When the Prometheus config generator runs
Then a second scrape job for metrics is included (FR35)

Given the Prometheus config
When the Overview dashboard queries `up{job="health"}`
Then it can display the agent status badge (FR32, FR34)

Given the deploy package
When I inspect static/alert_rules.yml
Then it is a static embedded file (not a template)

Given the deploy package test suite
When tests run
Then consistency_test.go verifies that all static/ and templates/ files referenced in code actually exist as embedded assets
```

## Tasks / Subtasks

- [x] **Task 1: Create templates/prometheus.yml.tmpl** — Prometheus config template with 2 scrape jobs
- [x] **Task 2: Create static/alert_rules.yml** — Static alert rules file (Prometheus syntax, NOT Go template)
- [x] **Task 3: Update embed.go** — Add staticFS for static/* directory
- [x] **Task 4: Create prometheus.go** — GeneratePrometheus + RenderPrometheus + CopyAlertRules
- [x] **Task 5: Create prometheus_test.go** — 12 tests + 2 golden files
- [x] **Task 6: Create consistency_test.go** — Verify embedded assets exist (3 templates + 1 static)
- [x] **Task 7: Verify lint + test pass** — 171 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **prometheus.yml.tmpl** uses `{{.JobHealth}}` and `{{.JobMetrics}}` methods on TemplateContext that return constants.
2. **alert_rules.yml** is a static file (NOT Go template) — avoids conflict with Prometheus `{{ }}` syntax.
3. **CopyAlertRules()** reads from `staticFS` embed and writes to `.volra/alert_rules.yml`.
4. **embed.go** updated with `//go:embed static/*` for `staticFS`.
5. **TemplateContext** gained `JobHealth()` and `JobMetrics()` methods to expose constants to templates.
6. **consistency_test.go** validates all 3 templates and 1 static file exist in embedded FS.
7. **AgentDown alert** fires when `up{job="agent-health"} == 0` for 1 minute (severity: critical).

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Added JobHealth()/JobMetrics() methods to TemplateContext | Go templates can't access package constants directly |
| 2 | alert_rules.yml as static (not template) | Prometheus uses `{{ }}` syntax that conflicts with Go text/template |
| 3 | Re-added staticFS embed in embed.go | Was removed in Story 3.1 due to unused lint; now needed |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/templates/prometheus.yml.tmpl` | Created | Prometheus config template |
| `internal/deploy/static/alert_rules.yml` | Created | Static alert rules (AgentDown) |
| `internal/deploy/embed.go` | Modified | Added staticFS for static/* |
| `internal/deploy/context.go` | Modified | Added JobHealth()/JobMetrics() methods |
| `internal/deploy/prometheus.go` | Created | GeneratePrometheus + RenderPrometheus + CopyAlertRules |
| `internal/deploy/prometheus_test.go` | Created | 12 tests + 2 golden files |
| `internal/deploy/consistency_test.go` | Created | Embedded asset verification |
| `internal/deploy/testdata/golden/prometheus_minimal.golden` | Created | Golden file |
| `internal/deploy/testdata/golden/prometheus_custom_port.golden` | Created | Golden file |
