#!/bin/sh
# Volra installer — https://github.com/romerox3/volra
# Usage: curl -fsSL https://raw.githubusercontent.com/romerox3/volra/main/install.sh | sh
set -e

REPO="romerox3/volra"
BINARY="volra"
INSTALL_DIR="/usr/local/bin"

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux" ;;
        *)      echo "Error: unsupported OS: $OS"; exit 1 ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        arm64|aarch64) ARCH="arm64" ;;
        x86_64)        ARCH="amd64" ;;
        *)             echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

# Get latest release tag
get_latest_version() {
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
    if [ -z "$VERSION" ]; then
        echo "Error: could not determine latest version"
        exit 1
    fi
}

# Download and verify binary
download() {
    FILENAME="${BINARY}-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/SHA256SUMS"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
    curl -fsSL -o "${TMPDIR}/${FILENAME}" "$URL"
    curl -fsSL -o "${TMPDIR}/SHA256SUMS" "$CHECKSUMS_URL"

    # Verify checksum
    echo "Verifying checksum..."
    cd "$TMPDIR"
    if command -v sha256sum > /dev/null 2>&1; then
        grep "$FILENAME" SHA256SUMS | sha256sum -c - > /dev/null 2>&1
    elif command -v shasum > /dev/null 2>&1; then
        grep "$FILENAME" SHA256SUMS | shasum -a 256 -c - > /dev/null 2>&1
    else
        echo "Warning: no checksum tool found, skipping verification"
    fi
    cd - > /dev/null

    chmod +x "${TMPDIR}/${FILENAME}"
    DOWNLOADED="${TMPDIR}/${FILENAME}"
}

# Install binary
install_binary() {
    if [ -w "$INSTALL_DIR" ]; then
        mv "$DOWNLOADED" "${INSTALL_DIR}/${BINARY}"
    elif command -v sudo > /dev/null 2>&1; then
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$DOWNLOADED" "${INSTALL_DIR}/${BINARY}"
    else
        ALT_DIR="${HOME}/.local/bin"
        mkdir -p "$ALT_DIR"
        mv "$DOWNLOADED" "${ALT_DIR}/${BINARY}"
        echo ""
        echo "Installed to ${ALT_DIR}/${BINARY}"
        echo "Add to PATH if not already configured:"
        echo "  export PATH=\"${ALT_DIR}:\$PATH\""
        echo ""
        echo "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
        return
    fi
    echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
}

main() {
    detect_os
    detect_arch
    get_latest_version
    download
    install_binary
    echo ""
    echo "Run '${BINARY} --version' to verify installation."
}

main
