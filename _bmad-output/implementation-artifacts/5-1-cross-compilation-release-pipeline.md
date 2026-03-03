# Story 5.1: Cross-Compilation & Release Pipeline

Status: done

## Tasks / Subtasks

- [x] **Task 1: Create .github/workflows/ci.yml** — Build + test + lint + shellcheck
- [x] **Task 2: Create .github/workflows/release.yml** — Cross-compile + GitHub Release via softprops/action-gh-release

## Dev Agent Record

### File List

| File | Action | Purpose |
|------|--------|---------|
| `.github/workflows/ci.yml` | Created | CI pipeline: build, test -race, golangci-lint, shellcheck |
| `.github/workflows/release.yml` | Created | Release: cross-compile, checksums, GitHub Release |
