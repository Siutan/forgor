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

# Compare two semantic versions
# Returns: 0 if equal, 1 if first > second, 2 if first < second
compare_versions() {
    local ver1="$1"
    local ver2="$2"
    
    # Remove 'v' prefix if present
    ver1="${ver1#v}"
    ver2="${ver2#v}"
    
    # If versions are identical
    if [ "$ver1" = "$ver2" ]; then
        return 0
    fi
    
    # Split versions into arrays and compare
    local IFS='.'
    local ver1_array=($ver1)
    local ver2_array=($ver2)
    
    # Pad arrays to same length
    local max_len=${#ver1_array[@]}
    if [ ${#ver2_array[@]} -gt $max_len ]; then
        max_len=${#ver2_array[@]}
    fi
    
    for ((i=0; i<max_len; i++)); do
        local v1_part=${ver1_array[i]:-0}
        local v2_part=${ver2_array[i]:-0}
        
        # Remove non-numeric suffixes (like -rc1, -beta, etc.)
        v1_part=$(echo "$v1_part" | sed 's/[^0-9].*//')
        v2_part=$(echo "$v2_part" | sed 's/[^0-9].*//')
        
        # Default to 0 if empty
        v1_part=${v1_part:-0}
        v2_part=${v2_part:-0}
        
        if [ "$v1_part" -gt "$v2_part" ]; then
            return 1  # ver1 > ver2
        elif [ "$v1_part" -lt "$v2_part" ]; then
            return 2  # ver1 < ver2
        fi
    done
    
    return 0  # versions are equal
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

# Check if already installed and compare versions
check_existing() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version latest_version
        
        # Get current version (extract version number from output)
        current_version=$($BINARY_NAME --version 2>/dev/null | head -n1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        
        # Get latest version
        latest_version=$(get_latest_version)
        
        if [ "$current_version" = "unknown" ]; then
            warn "$BINARY_NAME is already installed but version could not be determined"
            echo -n "Do you want to reinstall? [y/N]: "
            read -r response
            case "$response" in
                [yY][eE][sS]|[yY]) 
                    log "Proceeding with reinstallation..."
                    return 0
                    ;;
                *)
                    log "Installation cancelled."
                    exit 0
                    ;;
            esac
        fi
        
        log "Current version: $current_version"
        log "Latest version: $latest_version"
        
        # Compare versions
        compare_versions "$latest_version" "$current_version"
        local comparison=$?
        
        case $comparison in
            0)
                success "$BINARY_NAME is already up to date (version $current_version)"
                exit 0
                ;;
            1)
                log "A newer version ($latest_version) is available!"
                echo -n "Do you want to update from $current_version to $latest_version? [Y/n]: "
                read -r response
                case "$response" in
                    [nN][oO]|[nN])
                        log "Update cancelled."
                        exit 0
                        ;;
                    *)
                        log "Proceeding with update..."
                        return 0
                        ;;
                esac
                ;;
            2)
                warn "Current version ($current_version) is newer than latest release ($latest_version)"
                echo -n "Do you want to downgrade to $latest_version? [y/N]: "
                read -r response
                case "$response" in
                    [yY][eE][sS]|[yY])
                        log "Proceeding with downgrade..."
                        return 0
                        ;;
                                         *)
                         log "Installation cancelled."
                         exit 0
                         ;;
                 esac
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