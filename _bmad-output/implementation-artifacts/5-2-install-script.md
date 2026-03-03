# Story 5.2: Install Script

Status: done

## Tasks / Subtasks

- [x] **Task 1: Create install.sh** — POSIX-compatible script with OS/arch detection, SHA256 verification, sudo fallback

## Dev Agent Record

### Completion Notes

1. Detects OS via `uname -s` (darwin/linux), arch via `uname -m` (arm64/amd64).
2. Downloads binary + SHA256SUMS from GitHub Releases.
3. Verifies checksum with `sha256sum` or `shasum -a 256`.
4. Falls back to `~/.local/bin` if `/usr/local/bin` not writable and no sudo.
5. Provides PATH configuration instructions for the fallback case.

### File List

| File | Action | Purpose |
|------|--------|---------|
| `install.sh` | Created | POSIX install script with checksum verification |
