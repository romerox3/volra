# Party Mode Consensus — Limitaciones Volra v2.0

Resultados de 3 debates multi-agente BMAD-S (Frink/Lisa/Homer) sobre las limitaciones encontradas en E2E.

> **Estado v2.1 (2026-03-04)**: Todas las 7 limitaciones han sido **IMPLEMENTADAS** en Epic 10 (v2.1). Las decisiones de Party Mode se siguieron fielmente en la implementación.

---

## Debate 1: Service Reliability (L2 + L3) — IMPLEMENTADO v2.1

### Decisión: Smart Defaults con Override

Healthchecks automáticos para imágenes conocidas, override manual en Agentfile, opción de desactivar.

**Mapa de healthchecks conocidos** (implementado en `internal/deploy/service_defaults.go`):

| Imagen prefix | Test command | Interval | Start period |
|---------------|-------------|----------|-------------|
| `postgres` | `pg_isready -U postgres \|\| exit 1` | 5s | 10s |
| `redis` | `redis-cli ping \| grep -q PONG` | 5s | 5s |
| `chromadb/chroma` | `curl -sf http://localhost:8000/api/v1/heartbeat \|\| exit 1` | 5s | 15s |

**Cambios implementados**:

1. `HealthcheckConfig` struct en `internal/agentfile/agentfile.go`
2. `DeployHealthcheck` + campo en `ServiceContext` en `internal/deploy/context.go`
3. `resolveHealthcheck()` en `internal/deploy/service_defaults.go` — cadena: explicit > auto por imagen > nil
4. Template `healthcheck:` block con `toJSON` para test array
5. Template `depends_on` como mapa con `condition: service_healthy` / `service_started`
6. `toJSON` template func registrada en `internal/deploy/compose.go`

---

## Debate 2: Ports & Resources (L5 + L6 + H2) — IMPLEMENTADO v2.1

### Decisión: Separar container port de host port + resource limits

**Campos implementados** (en `internal/agentfile/agentfile.go`):

```go
// En Agentfile
HostPort int `yaml:"host_port,omitempty"` // default = port (backward compat)

// En Service
HostPort  int             `yaml:"host_port,omitempty"` // default = 0 (NO exponer)
Resources *ResourceConfig `yaml:"resources,omitempty"`

// Nuevo tipo
type ResourceConfig struct {
    MemLimit string `yaml:"mem_limit,omitempty"`
    CPUs     string `yaml:"cpus,omitempty"`
}
```

**Defaults de recursos** (implementado en `internal/deploy/service_defaults.go`):

| Servicio | mem_limit | cpus |
|----------|-----------|------|
| Redis | 256m | 0.25 |
| PostgreSQL | 512m | 0.5 |
| ChromaDB | 1g | 1.0 |

> Nota: Agent resources NO se aplican auto (quedó fuera de scope — solo services).

**Puertos de infra**: `AgentHostPort`, `PrometheusHostPort`, `GrafanaHostPort` derivados en `internal/deploy/context.go`. Deploy summary en `deploy.go` usa puertos resueltos.

**Validación**: `validateHostPort()` en `internal/agentfile/validate.go` — rango 1-65535, detección de conflictos entre services.

---

## Debate 3: Security & ML (L1 + L4 + L7 + H3) — IMPLEMENTADO v2.1

### Decisiones por limitación:

#### H3 — Build-time model downloads (PRIORIDAD P0) — IMPLEMENTADO

`BuildConfig` struct en `internal/agentfile/agentfile.go`:

```yaml
build:
  setup_commands:
    - "python -m nltk.downloader punkt"
    - "python -c \"import transformers; transformers.AutoTokenizer.from_pretrained('bert-base-uncased')\""
  cache_dirs:
    - /root/nltk_data
    - /root/.cache/huggingface
```

- Template `Dockerfile.tmpl`: `RUN` commands after pip install, `COPY --from=builder` for cache dirs
- Validación en `validateBuild()`: solo con `dockerfile: auto`, max 20 commands, cache_dirs absolutos
- Golden file: `dockerfile_build.golden`

#### L7 — Secrets separation (PRIORIDAD P1) — IMPLEMENTADO

Implementación en `internal/deploy/envfiles.go` (NEW):
- `GenerateEnvFiles()` genera `agent.env` + `{name}-{service}.env` desde `.env` fuente
- Permisos 0600 en archivos generados
- Template usa `./agent.env` y `./name-service.env`
- NO requiere cambio en schema Agentfile — separación es transparente basada en `env:` existente

> Nota: La propuesta original de Party Mode sugería cambiar el schema de `env:` a un mapa `agent/services`. La implementación final mantiene el schema actual sin cambios — la separación ocurre en generación, no en parsing.

#### L4 — GPU pre-flight check (PRIORIDAD P2) — IMPLEMENTADO

`CheckGPUAvailable()` en `internal/deploy/preflight.go` (NEW):
- `docker info --format '{{.Runtimes}}'` + check "nvidia"
- Error codes: E307 (CodeGPUNotAvailable), E308 (CodeGPUCheckFailed)
- Ejecutado en `deploy.go` ANTES de Orchestrate

#### L1 — read_only + tmpfs auto (PRIORIDAD P3) — IMPLEMENTADO

`TmpfsMount` struct + auto-inject en `internal/deploy/context.go`:
- Cuando `read_only: true` y `len(security.Tmpfs) == 0`: auto-inyecta `/tmp` (100M) + `/app/__pycache__` (50M)
- Override manual con `security.tmpfs[]` en Agentfile
- Template `docker-compose.yml.tmpl`: bloque `tmpfs:` condicional

---

## Priorización Final Consolidada

| Prio | Limitación | Categoría | Impacto | Esfuerzo | **Estado** |
|------|-----------|-----------|---------|----------|------------|
| **P0** | L2+L3: Service healthchecks + depends_on | Reliability | ALTO — elimina race conditions | 1 día | **IMPLEMENTADO** |
| **P0** | H3: Build-time model downloads | ML support | ALTO — habilita ML con auto dockerfile | 2-3 días | **IMPLEMENTADO** |
| **P1** | L7: Secrets separation | Security | ALTO — elimina cross-contamination | 1-2 días | **IMPLEMENTADO** |
| **P1** | L5+H2: Port separation | Usability | MEDIO — habilita multi-agent | 2 días | **IMPLEMENTADO** |
| **P2** | L6: Resource limits | Robustness | MEDIO — previene OOM | 0.5 días | **IMPLEMENTADO** |
| **P2** | L4: GPU pre-flight | DX | MEDIO — error temprano | 0.5 días | **IMPLEMENTADO** |
| **P3** | L1: read_only + tmpfs | Security | BAJO — solo Linux prod | 1 día | **IMPLEMENTADO** |

**Total implementado**: 7/7 limitaciones resueltas en Epic 10 (v2.1), sprint único.

---

## Backward Compatibility — VERIFICADA

Todos los cambios son **backward compatible** (confirmado con test suite completa):
- Campos nuevos son `omitempty` — Agentfiles v1.0/v1.1/v1.2/v2.0 siguen funcionando sin cambios
- `host_port` default = `port` para el agente (misma behavior)
- `build`, `resources`, `tmpfs`, `healthcheck` son opcionales
- Healthchecks auto se generan sin cambios en Agentfile del usuario
- Separación de env es transparente (Volra genera los archivos desde `.env` existente)
- Tests con `-race -count=1` pasan al 100%

---

## Archivos Nuevos/Modificados (v2.1)

| Archivo | Tipo |
|---------|------|
| `internal/deploy/service_defaults.go` | NUEVO — knownHealthchecks + knownResources + resolve functions |
| `internal/deploy/envfiles.go` | NUEVO — per-service env file generator |
| `internal/deploy/preflight.go` | NUEVO — GPU pre-flight check |
| `internal/deploy/service_defaults_test.go` | NUEVO — 12 tests |
| `internal/deploy/envfiles_test.go` | NUEVO — 6 tests |
| `internal/deploy/preflight_test.go` | NUEVO — 3 tests |
| `internal/agentfile/agentfile.go` | MOD — 5 new structs/fields |
| `internal/agentfile/validate.go` | MOD — validateBuild + validateHostPort |
| `internal/deploy/context.go` | MOD — ServiceContext + TemplateContext extensions |
| `internal/deploy/compose.go` | MOD — toJSON + gt template funcs |
| `internal/deploy/deploy.go` | MOD — GPU check + env files + port-aware summary |
| `internal/deploy/templates/docker-compose.yml.tmpl` | MOD — all 7 features |
| `internal/deploy/templates/Dockerfile.tmpl` | MOD — setup_commands + cache_dirs |
| `internal/output/catalog.go` | MOD — E307/E308 error codes |
| `tests/e2e/e2e_test.go` | MOD — 6 new Phase 2 tests |
