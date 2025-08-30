#!/bin/bash

# Docker Compose MCP Server Installation Script
set -e

# Configuration
BINARY_NAME="docker-compose-mcp"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/docker-compose-mcp"
GITHUB_REPO="[username]/docker-compose-mcp"  # Replace with actual repo
LATEST_RELEASE_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect platform
detect_platform() {
    local os=""
    local arch=""
    
    case "$(uname -s)" in
        Darwin*)
            os="darwin"
            ;;
        Linux*)
            os="linux"
            ;;
        MINGW*|CYGWIN*|MSYS*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}/${arch}"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        log_error "Neither curl nor wget is available"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_warning "Docker is not installed or not in PATH"
        log_warning "Please ensure Docker is available before using docker-compose-mcp"
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
        log_warning "Docker Compose is not available"
        log_warning "Please ensure Docker Compose is installed before using docker-compose-mcp"
    fi
    
    log_success "Dependencies check completed"
}

# Download file
download_file() {
    local url=$1
    local output=$2
    
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$output"
    else
        log_error "No download tool available"
        exit 1
    fi
}

# Get latest release info
get_latest_release() {
    log_info "Fetching latest release information..."
    
    local temp_file=$(mktemp)
    
    if ! download_file "$LATEST_RELEASE_URL" "$temp_file"; then
        log_error "Failed to fetch release information"
        exit 1
    fi
    
    # Extract version and download URL
    local version=$(grep '"tag_name"' "$temp_file" | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    local platform=$(detect_platform)
    local os=${platform%/*}
    local arch=${platform#*/}
    
    local asset_name="${BINARY_NAME}-${version}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        asset_name="${asset_name}.zip"
    else
        asset_name="${asset_name}.tar.gz"
    fi
    
    local download_url=$(grep "browser_download_url.*${asset_name}" "$temp_file" | sed -E 's/.*"browser_download_url": "([^"]+)".*/\1/')
    
    rm "$temp_file"
    
    if [ -z "$version" ] || [ -z "$download_url" ]; then
        log_error "Failed to parse release information"
        exit 1
    fi
    
    echo "$version $download_url $asset_name"
}

# Install binary
install_binary() {
    local version=$1
    local download_url=$2
    local asset_name=$3
    
    log_info "Installing docker-compose-mcp version $version..."
    
    # Create temporary directory
    local temp_dir=$(mktemp -d)
    local archive_path="$temp_dir/$asset_name"
    
    # Download archive
    log_info "Downloading $asset_name..."
    if ! download_file "$download_url" "$archive_path"; then
        log_error "Failed to download $asset_name"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Extract archive
    log_info "Extracting archive..."
    cd "$temp_dir"
    
    if [[ "$asset_name" == *.zip ]]; then
        if ! command -v unzip &> /dev/null; then
            log_error "unzip is required to extract Windows archives"
            rm -rf "$temp_dir"
            exit 1
        fi
        unzip -q "$asset_name"
    else
        tar -xzf "$asset_name"
    fi
    
    # Find binary
    local binary_path=""
    if [ -f "${BINARY_NAME}" ]; then
        binary_path="${BINARY_NAME}"
    elif [ -f "${BINARY_NAME}.exe" ]; then
        binary_path="${BINARY_NAME}.exe"
    else
        # Look in subdirectories
        binary_path=$(find . -name "${BINARY_NAME}" -o -name "${BINARY_NAME}.exe" | head -1)
    fi
    
    if [ -z "$binary_path" ]; then
        log_error "Binary not found in archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Install binary
    log_info "Installing binary to $INSTALL_DIR..."
    
    if [ ! -w "$INSTALL_DIR" ]; then
        log_info "Installing with sudo (admin privileges required)..."
        sudo cp "$binary_path" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        cp "$binary_path" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Cleanup
    rm -rf "$temp_dir"
    
    log_success "Binary installed successfully"
}

# Create configuration directory
create_config_dir() {
    log_info "Creating configuration directory..."
    
    mkdir -p "$CONFIG_DIR"
    
    # Create sample configuration
    cat > "$CONFIG_DIR/config.env" << 'EOF'
# Docker Compose MCP Server Configuration
# Uncomment and modify values as needed

# Logging
# MCP_LOG_LEVEL=info
# MCP_LOG_FORMAT=text
# MCP_LOG_FILE=

# Performance
# MCP_ENABLE_CACHE=true
# MCP_CACHE_SIZE=100
# MCP_CACHE_MAX_AGE=30m
# MCP_ENABLE_METRICS=true
# MCP_ENABLE_PARALLEL=true
# MCP_MAX_WORKERS=4

# Timeouts
# MCP_COMMAND_TIMEOUT=5m
# MCP_SHUTDOWN_TIMEOUT=30s
# MCP_SESSION_TIMEOUT=1h

# Docker
# COMPOSE_FILE=docker-compose.yml
# COMPOSE_PROJECT_NAME=
# DOCKER_HOST=
EOF
    
    log_success "Configuration directory created at $CONFIG_DIR"
}

# Generate Claude Desktop config
generate_claude_config() {
    local config_file="$HOME/.config/Claude/claude_desktop_config.json"
    local config_dir=$(dirname "$config_file")
    
    log_info "Generating Claude Desktop configuration..."
    
    mkdir -p "$config_dir"
    
    if [ -f "$config_file" ]; then
        log_info "Backing up existing configuration..."
        cp "$config_file" "${config_file}.backup.$(date +%s)"
    fi
    
    # Check if MCP servers section exists
    if [ -f "$config_file" ] && grep -q '"mcpServers"' "$config_file"; then
        log_info "Adding docker-compose-mcp to existing Claude Desktop configuration..."
        # Would need jq for proper JSON manipulation - for now, just inform user
        log_warning "Please manually add docker-compose-mcp to your existing Claude Desktop configuration"
        log_info "Configuration template available at: docs/CLAUDE_DESKTOP_INTEGRATION.md"
    else
        log_info "Creating new Claude Desktop configuration..."
        cat > "$config_file" << EOF
{
  "mcpServers": {
    "docker-compose-mcp": {
      "command": "$INSTALL_DIR/$BINARY_NAME",
      "args": [],
      "env": {
        "MCP_LOG_LEVEL": "info",
        "MCP_LOG_FORMAT": "text",
        "MCP_ENABLE_CACHE": "true",
        "MCP_ENABLE_METRICS": "true",
        "MCP_ENABLE_PARALLEL": "true",
        "MCP_MAX_WORKERS": "4",
        "MCP_COMMAND_TIMEOUT": "5m",
        "MCP_SHUTDOWN_TIMEOUT": "30s"
      }
    }
  }
}
EOF
        log_success "Claude Desktop configuration created"
    fi
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        log_error "Binary not found in PATH"
        log_info "You may need to restart your shell or add $INSTALL_DIR to your PATH"
        return 1
    fi
    
    # Test binary
    if ! "$BINARY_NAME" --version &> /dev/null 2>&1; then
        log_warning "Binary installed but --version flag not working (this is expected for MCP servers)"
    fi
    
    log_success "Installation verified successfully"
    return 0
}

# Uninstall
uninstall() {
    log_info "Uninstalling docker-compose-mcp..."
    
    # Remove binary
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        if [ ! -w "$INSTALL_DIR" ]; then
            sudo rm "$INSTALL_DIR/$BINARY_NAME"
        else
            rm "$INSTALL_DIR/$BINARY_NAME"
        fi
        log_success "Binary removed"
    else
        log_warning "Binary not found at $INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Ask about configuration
    read -p "Remove configuration directory $CONFIG_DIR? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        log_success "Configuration directory removed"
    fi
    
    log_info "Uninstall completed"
    log_info "Note: Claude Desktop configuration not modified automatically"
}

# Show help
show_help() {
    cat << EOF
Docker Compose MCP Server Installation Script

USAGE:
    $0 [COMMAND] [OPTIONS]

COMMANDS:
    install     Install the latest version (default)
    uninstall   Remove docker-compose-mcp
    update      Update to latest version
    help        Show this help message

OPTIONS:
    --version VERSION    Install specific version
    --no-config         Skip configuration setup
    --no-claude-config  Skip Claude Desktop configuration

EXAMPLES:
    $0                          # Install latest version
    $0 install                  # Install latest version
    $0 --version v1.0.0         # Install specific version
    $0 uninstall                # Remove installation
    $0 update                   # Update to latest

DIRECTORIES:
    Binary: $INSTALL_DIR/$BINARY_NAME
    Config: $CONFIG_DIR/
    Claude: ~/.config/Claude/claude_desktop_config.json

For more information, visit: https://github.com/${GITHUB_REPO}
EOF
}

# Main function
main() {
    local command="install"
    local specific_version=""
    local skip_config=false
    local skip_claude_config=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            install|uninstall|update|help)
                command=$1
                shift
                ;;
            --version)
                specific_version=$2
                shift 2
                ;;
            --no-config)
                skip_config=true
                shift
                ;;
            --no-claude-config)
                skip_claude_config=true
                shift
                ;;
            -h|--help)
                command="help"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    case $command in
        help)
            show_help
            ;;
        uninstall)
            uninstall
            ;;
        install|update)
            log_info "Starting installation process..."
            
            check_dependencies
            
            if [ -n "$specific_version" ]; then
                log_error "Specific version installation not implemented yet"
                log_info "Please install from source or use latest release"
                exit 1
            fi
            
            # Get release info and install
            local release_info
            release_info=$(get_latest_release)
            read -r version download_url asset_name <<< "$release_info"
            
            install_binary "$version" "$download_url" "$asset_name"
            
            if [ "$skip_config" != "true" ]; then
                create_config_dir
            fi
            
            if [ "$skip_claude_config" != "true" ]; then
                generate_claude_config
            fi
            
            verify_installation
            
            log_success "Installation completed successfully!"
            log_info ""
            log_info "Next steps:"
            log_info "1. Restart Claude Desktop if it's running"
            log_info "2. Navigate to a project with docker-compose.yml"
            log_info "3. Ask Claude to help with Docker Compose operations"
            log_info ""
            log_info "Documentation: docs/CLAUDE_DESKTOP_INTEGRATION.md"
            log_info "Configuration: $CONFIG_DIR/config.env"
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"