#!/bin/bash

# Install script for forgor
# Usage: curl -fsSL https://raw.githubusercontent.com/Siutan/forgor/main/install.sh | sh

set -e

# Configuration
REPO="Siutan/forgor"  # Replace with your GitHub username/repo
BINARY_NAME="forgor"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Linux*)  os="Linux" ;;
        Darwin*) os="Darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="Windows" ;; # we actually don't support Windows but i cbs rn
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l|armv6l) arch="arm" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
    
    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    if [ -z "$version" ]; then
        error "Failed to get latest version"
    fi
    echo "$version"
}

# Download and install binary
install_binary() {
    local version platform tmpdir filename url
    
    version=$(get_latest_version)
    platform=$(detect_platform)
    tmpdir=$(mktemp -d)
    
    log "Latest version: $version"
    log "Platform: $platform"
    log "Installing to: $INSTALL_DIR"
    
    # Determine file extension
    if [[ "$platform" == "windows-"* ]]; then
        filename="${BINARY_NAME}_${platform}.zip"
    else
        filename="${BINARY_NAME}_${platform}.tar.gz"
    fi
    
    url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    
    log "Downloading from: $url"
    
    # Download
    if ! curl -fsSL "$url" -o "${tmpdir}/${filename}"; then
        error "Failed to download $filename"
    fi
    
    # Extract
    cd "$tmpdir"
    if [[ "$filename" == *.zip ]]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$filename"
        else
            error "unzip is required but not installed"
        fi
    else
        if command -v tar >/dev/null 2>&1; then
            tar -xzf "$filename"
        else
            error "tar is required but not installed"
        fi
    fi
    
    # Find binary
    local binary_path
    if [[ "$platform" == "windows_"* ]]; then
        binary_path="${BINARY_NAME}.exe"
    else
        binary_path="${BINARY_NAME}"
    fi
    
    if [ ! -f "$binary_path" ]; then
        error "Binary not found in archive"
    fi
    
    # Install binary
    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary_path" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$binary_path"
    else
        log "Installing to $INSTALL_DIR requires sudo"
        sudo mv "$binary_path" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$binary_path"
    fi
    
    # Cleanup
    cd /
    rm -rf "$tmpdir"
    
    success "$BINARY_NAME $version installed successfully!"
}

# Check if already installed
check_existing() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version
        current_version=$($BINARY_NAME --version 2>/dev/null | head -n1 || echo "unknown")
        warn "$BINARY_NAME is already installed: $current_version"
        echo -n "Do you want to reinstall? [y/N]: "
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY]) 
                log "Proceeding with reinstallation..."
                ;;
            *)
                log "Installation cancelled."
                exit 0
                ;;
        esac
    fi
}

# Main installation flow
main() {
    log "Starting installation of $BINARY_NAME"
    
    # Check dependencies
    for cmd in curl; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "$cmd is required but not installed"
        fi
    done
    
    check_existing
    install_binary
    
    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        success "Installation verified! Try running: $BINARY_NAME --help"
    else
        warn "Installation complete, but $BINARY_NAME is not in PATH"
        warn "You may need to add $INSTALL_DIR to your PATH"
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --version     Show version info"
        echo ""
        echo "Examples:"
        echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sh"
        echo "  wget -qO- https://raw.githubusercontent.com/$REPO/main/install.sh | sh"
        exit 0
        ;;
    --version)
        echo "forgor installer v1.0.0"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac