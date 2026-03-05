# E2E Test Findings — Volra v2.0

Documento de hallazgos, problemas detectados y comportamientos observados durante la validación E2E con 8 agentes Python reales.

> **Actualización v2.1 (2026-03-04)**: Todas las limitaciones L1-L7 y hallazgos H2-H3 han sido **RESUELTAS** en la implementación v2.1 (Epic 10). Ver sección de cada limitación para detalles.

## Ejecución Final

- **Fecha**: 2026-03-04
- **Docker**: 29.1.3
- **Go**: 1.25.0
- **Plataforma**: darwin/arm64
- **Resultado**: 27/27 PASS (49.5s) + 6 nuevos tests Phase 2 (v2.1)

---

## Resumen de Resultados

| Phase | Tests | Resultado | Tiempo |
|-------|-------|-----------|--------|
| Phase 1 — Agentfile Load | 15/15 | PASS | <1s |
| Phase 2 — Compose Generation | 14/14 | PASS | <1s |
| Phase 3 — Deploy + Health | 6/6 | PASS | 35.6s |
| Phase 4 — Expected Failures | 2/2 | PASS | 13.5s |

> Phase 2 ampliada de 8 a 14 tests en v2.1 con: ComposeHealthchecks, ComposeDependsOnHealthy, ComposeServicePortsNotExposed, ComposeAutoResources, ComposeTmpfs, ComposeEnvFileSeparation.

### Phase 3 — Deploy Times

| Agent | Resultado | Tiempo | Services |
|-------|-----------|--------|----------|
| A1 echo-agent | PASS | 5.7s | 0 |
| A2 sentiment-analyzer | PASS | 6.4s | 0 |
| A3 doc-summarizer | PASS | 4.3s | 0 |
| A4 rag-kb | PASS | 4.3s | 1 (Redis) |
| A5 conv-agent | PASS | 6.9s | 2 (Redis + PG) |
| A8 orchestrator | PASS | 8.0s | 3 (Redis + PG + ChromaDB) |

---

## Limitaciones Confirmadas del Sistema

### L1: `read_only: true` + Python — RESUELTA en v2.1

- **Status**: ~~NO CONFIRMADA~~ → **RESUELTA**
- **Agente**: A6 (ai-gateway)
- **Observación original**: El agent arranca en macOS pero podría fallar en Linux por `__pycache__` y `/tmp`
- **Resolución v2.1**: Auto-inyección de `tmpfs` para `/tmp` (100M) y `/app/__pycache__` (50M) cuando `read_only: true` y no hay `tmpfs` explícito. Nuevo struct `TmpfsMount` en `SecurityContext`. Override manual con `security.tmpfs[]` en Agentfile.
- **Archivos**: `agentfile.go` (TmpfsMount), `context.go` (auto-inject), `docker-compose.yml.tmpl` (tmpfs block)
- **E2E**: `TestPhase2_ComposeTmpfs`

### L2: Sin `depends_on` con health condition — RESUELTA en v2.1

- **Status**: ~~MITIGADA POR RETRY~~ → **RESUELTA**
- **Agentes**: A5, A8 (usan Postgres)
- **Observación original**: Race condition mitigada por retry loops en código Python
- **Resolución v2.1**: `depends_on` genera `condition: service_healthy` cuando el service tiene healthcheck (auto o custom), `condition: service_started` si no tiene. Formato cambiado de lista a mapa.
- **Archivos**: `docker-compose.yml.tmpl` (depends_on format), `context.go` (healthcheck resolution)
- **E2E**: `TestPhase2_ComposeDependsOnHealthy`

### L3: Sin healthcheck para services — RESUELTA en v2.1

- **Status**: ~~CONFIRMADA~~ → **RESUELTA**
- **Observación original**: Template no generaba `healthcheck:` para Redis, PG, ChromaDB
- **Resolución v2.1**: Mapa `knownHealthchecks` en `service_defaults.go` auto-resuelve healthchecks para imágenes conocidas (postgres: `pg_isready`, redis: `redis-cli ping`, chromadb/chroma: `curl heartbeat`). Override con `healthcheck:` custom en Service del Agentfile. Función `resolveHealthcheck()` con cadena: explicit > auto > nil.
- **Archivos**: `service_defaults.go` (NEW), `context.go` (resolución), `docker-compose.yml.tmpl` (healthcheck block)
- **E2E**: `TestPhase2_ComposeHealthchecks`

### L4: GPU compose falla sin NVIDIA runtime — RESUELTA en v2.1

- **Status**: ~~PARCIAL~~ → **RESUELTA**
- **Agente**: A7 (vision-classifier)
- **Observación original**: `docker compose config` no detecta ausencia de NVIDIA runtime
- **Resolución v2.1**: `CheckGPUAvailable()` en `preflight.go` ejecuta `docker info --format '{{.Runtimes}}'` y verifica presencia de "nvidia" ANTES de deploy. Error codes E307 (GPU not available) y E308 (GPU check failed) con mensajes actionables.
- **Archivos**: `preflight.go` (NEW), `deploy.go` (check antes de Orchestrate), `catalog.go` (E307/E308)

### L5: Puerto 8000 conflicts entre agentes — RESUELTA en v2.1

- **Status**: ~~CONFIRMADA~~ → **RESUELTA**
- **Observación original**: 2 agentes en misma máquina imposibles sin cambiar ports
- **Resolución v2.1**: Nuevo campo `host_port` en Agentfile (default = `port` para backward compat) y en Service (default = 0, NO exponer al host). Validación de rango y conflictos. Template usa `AgentHostPort:Port` en vez de `Port:Port`.
- **Archivos**: `agentfile.go` (HostPort fields), `validate.go` (validateHostPort), `context.go` (port derivation), `docker-compose.yml.tmpl`
- **E2E**: `TestPhase2_ComposeServicePortsNotExposed`

### L6: Sin resource limits para services — RESUELTA en v2.1

- **Status**: ~~CONFIRMADA~~ → **RESUELTA**
- **Observación original**: Redis/PG/ChromaDB sin límites de memoria
- **Resolución v2.1**: `ResourceConfig` struct con `mem_limit` y `cpus`. Mapa `knownResources` en `service_defaults.go` con defaults (Redis: 256m/0.25, PG: 512m/0.5, ChromaDB: 1g/1.0). Override con `resources:` custom en Service. Template genera `deploy.resources.limits` block.
- **Archivos**: `agentfile.go` (ResourceConfig), `service_defaults.go` (defaults), `context.go` (resolución), `docker-compose.yml.tmpl` (limits block)
- **E2E**: `TestPhase2_ComposeAutoResources`

### L7: Secrets en `.env` plain text — RESUELTA en v2.1

- **Status**: ~~CONFIRMADA~~ → **RESUELTA**
- **Observación original**: `env_file: ../.env` carga todo el archivo a todos los containers
- **Resolución v2.1**: `GenerateEnvFiles()` en `envfiles.go` genera archivos separados: `agent.env` (solo vars del agent) + `{name}-{service}.env` (solo vars del service). Cada container solo recibe su archivo. Permisos 0600. Template usa `./agent.env` y `./name-service.env`.
- **Archivos**: `envfiles.go` (NEW), `deploy.go` (pipeline), `docker-compose.yml.tmpl` (env_file paths)
- **E2E**: `TestPhase2_ComposeEnvFileSeparation`

---

## Hallazgos Nuevos (no previstos en el plan)

### H1: Docker bind mount crea directorios para archivos inexistentes

- **Severidad**: MEDIA
- **Status**: PENDIENTE (no abordado en v2.1)
- **Descubierto**: Test Phase 3 generaba solo compose sin prometheus/blackbox/grafana → Docker creaba dirs basura
- **Impacto en Volra**: Si un usuario ejecuta `docker compose up` sin `volra deploy`, los archivos de infra no existen y Docker corrompe el directorio
- **Mejora Volra**: El compose generado debería verificar que los archivos referenciados existen, o documentar claramente que solo `volra deploy` es válido

### H2: Puertos de infra fijos (Prometheus 9090, Grafana 3001) — RESUELTA en v2.1

- **Severidad**: MEDIA
- **Status**: ~~PENDIENTE~~ → **RESUELTA** (junto con L5)
- **Observación original**: Infra de observabilidad usa puertos fijos
- **Resolución v2.1**: `PrometheusHostPort` y `GrafanaHostPort` derivados en `TemplateContext` y usados en template. Cuando `host_port` del agent cambia, los puertos de infra se ajustan. Deploy summary muestra URLs con puertos correctos.
- **Archivos**: `context.go` (PrometheusHostPort, GrafanaHostPort), `docker-compose.yml.tmpl`, `deploy.go` (summary URLs)

### H3: Auto-generated Dockerfile no soporta build-time model downloads — RESUELTA en v2.1

- **Severidad**: ALTA para ML agents
- **Status**: ~~PENDIENTE~~ → **RESUELTA**
- **Descubierto**: A2 con NLTK necesita descargar datos en build time
- **Resolución v2.1**: Nuevo struct `BuildConfig` con `setup_commands` (max 20, ejecutados DESPUÉS de pip install) y `cache_dirs` (paths absolutos, copiados del builder stage). Validación: solo con `dockerfile: auto`. Template Dockerfile.tmpl genera `RUN` para cada command y `COPY --from=builder` para cada cache dir.
- **Archivos**: `agentfile.go` (BuildConfig), `validate.go` (validateBuild), `Dockerfile.tmpl` (setup + cache)

### H4: Cleanup de tests requiere orden cuidadoso

- **Severidad**: BAJA (solo afecta tests)
- **Status**: RESUELTO en v2.0
- **Descubierto**: Si el cleanup de `docker compose down` se registra después del `require`, nunca se ejecuta al fallar → contenedores huérfanos → siguiente test falla por conflicto de puerto
- **Fix aplicado**: `t.Cleanup()` se registra ANTES del `docker compose up`

---

## Análisis de Feature Coverage Real

| Feature | Parsing | Compose Gen | Deploy | Health |
|---------|:-------:|:-----------:|:------:|:------:|
| generic | OK | OK | OK | OK |
| langgraph | OK | OK | OK | OK |
| dockerfile: auto | OK | OK | OK | OK |
| dockerfile: custom | OK | OK | OK | OK |
| health_timeout | OK | OK | N/A | OK |
| volumes | OK | OK | OK | OK |
| env | OK | OK | OK | OK |
| services | OK | OK | OK | OK |
| service.env | OK | OK | OK | OK |
| service.volumes | OK | OK | OK | OK |
| security | OK | OK | OK* | OK* |
| gpu | OK | OK | Skip | Skip |
| **healthcheck** (v2.1) | OK | OK | — | — |
| **host_port** (v2.1) | OK | OK | — | — |
| **build.setup_commands** (v2.1) | OK | OK | — | — |
| **resources** (v2.1) | OK | OK | — | — |
| **tmpfs** (v2.1) | OK | OK | — | — |
| **env separation** (v2.1) | OK | OK | — | — |
| **package_manager** (v0.2) | OK | OK | — | — |

\* A6 funciona en macOS; auto-tmpfs en v2.1 mitiga riesgo en Linux.

---

## Resumen de Mejoras — Estado Actual

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| 1 | `depends_on` con `condition: service_healthy` | ALTO | BAJO | **RESUELTO v2.1** |
| 2 | Healthchecks para services conocidos | ALTO | MEDIO | **RESUELTO v2.1** |
| 3 | Port separation (`host_port`) | MEDIO | MEDIO | **RESUELTO v2.1** |
| 4 | Resource limits para services | MEDIO | BAJO | **RESUELTO v2.1** |
| 5 | Pre-check NVIDIA runtime para GPU | MEDIO | BAJO | **RESUELTO v2.1** |
| 6 | Build-time `setup_commands` + `cache_dirs` | ALTO | MEDIO | **RESUELTO v2.1** |
| 7 | Per-service env file separation | MEDIO | ALTO | **RESUELTO v2.1** |
