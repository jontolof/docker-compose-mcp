#!/bin/bash

# Docker Compose MCP Server Build Script
set -e

# Configuration
BINARY_NAME="docker-compose-mcp"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')

# Build info
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.GoVersion=${GO_VERSION}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v git &> /dev/null; then
        log_warning "Git is not installed - version info will be limited"
    fi
    
    log_success "Dependencies check passed"
}

# Run tests
run_tests() {
    log_info "Running tests..."
    
    if ! go test ./... -v; then
        log_error "Tests failed"
        exit 1
    fi
    
    log_success "All tests passed"
}

# Run linting
run_lint() {
    log_info "Running linting..."
    
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./...
        log_success "Linting passed"
    else
        log_warning "golangci-lint not found, skipping linting"
        if command -v go &> /dev/null; then
            go vet ./...
            log_success "Go vet passed"
        fi
    fi
}

# Build single platform
build_platform() {
    local goos=$1
    local goarch=$2
    local output_name="${BINARY_NAME}"
    
    if [ "$goos" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="dist/${goos}-${goarch}/${output_name}"
    
    log_info "Building for ${goos}/${goarch}..."
    
    mkdir -p "dist/${goos}-${goarch}"
    
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags="${LDFLAGS} -s -w" \
        -o "$output_path" \
        cmd/server/main.go
    
    # Create checksum
    if command -v sha256sum &> /dev/null; then
        (cd "dist/${goos}-${goarch}" && sha256sum "$output_name" > "${output_name}.sha256")
    elif command -v shasum &> /dev/null; then
        (cd "dist/${goos}-${goarch}" && shasum -a 256 "$output_name" > "${output_name}.sha256")
    fi
    
    log_success "Built ${output_path}"
}

# Build all platforms
build_all() {
    log_info "Building for multiple platforms..."
    
    # Clean dist directory
    rm -rf dist
    mkdir -p dist
    
    # Build matrix
    declare -a platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
        "windows/arm64"
    )
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r goos goarch <<< "$platform"
        build_platform "$goos" "$goarch"
    done
    
    log_success "Built all platforms"
}

# Create release archives
create_archives() {
    log_info "Creating release archives..."
    
    cd dist
    
    for dir in */; do
        if [ -d "$dir" ]; then
            platform=${dir%/}
            archive_name="${BINARY_NAME}-${VERSION}-${platform}"
            
            if [[ "$platform" == *"windows"* ]]; then
                zip -r "${archive_name}.zip" "$dir"
            else
                tar -czf "${archive_name}.tar.gz" "$dir"
            fi
            
            log_success "Created archive for ${platform}"
        fi
    done
    
    cd ..
}

# Generate release notes
generate_release_notes() {
    log_info "Generating release notes..."
    
    cat > dist/RELEASE_NOTES.md << EOF
# Docker Compose MCP Server v${VERSION}

## Build Information
- **Version:** ${VERSION}
- **Build Time:** ${BUILD_TIME}
- **Git Commit:** ${GIT_COMMIT}
- **Go Version:** ${GO_VERSION}

## Installation

### Download Binary
Download the appropriate binary for your platform from the releases page.

### Build from Source
\`\`\`bash
git clone https://github.com/[username]/docker-compose-mcp.git
cd docker-compose-mcp
go build -o docker-compose-mcp cmd/server/main.go
\`\`\`

### Claude Desktop Integration
See [Claude Desktop Integration Guide](docs/CLAUDE_DESKTOP_INTEGRATION.md) for setup instructions.

## Features
- 14 comprehensive Docker Compose tools
- 90%+ output filtering for reduced context usage
- Smart caching and parallel execution
- Session-based monitoring for long-running operations
- Database migration and backup tools
- Comprehensive error handling and logging

## Checksums
All binaries include SHA256 checksums for verification.

## Support
For issues and support, please visit the GitHub repository.
EOF
    
    log_success "Generated release notes"
}

# Main build function
main() {
    local build_type=${1:-"single"}
    local target_os=${2:-$(go env GOOS)}
    local target_arch=${3:-$(go env GOARCH)}
    
    log_info "Starting build process..."
    log_info "Version: ${VERSION}"
    log_info "Build Time: ${BUILD_TIME}"
    log_info "Git Commit: ${GIT_COMMIT}"
    
    check_dependencies
    
    # Run tests unless explicitly disabled
    if [ "${SKIP_TESTS}" != "true" ]; then
        run_tests
    fi
    
    # Run linting unless explicitly disabled
    if [ "${SKIP_LINT}" != "true" ]; then
        run_lint
    fi
    
    case "$build_type" in
        "single")
            log_info "Building single platform: ${target_os}/${target_arch}"
            build_platform "$target_os" "$target_arch"
            ;;
        "all"|"release")
            build_all
            create_archives
            generate_release_notes
            ;;
        *)
            log_error "Unknown build type: $build_type"
            log_info "Usage: $0 [single|all|release] [os] [arch]"
            exit 1
            ;;
    esac
    
    log_success "Build process completed successfully!"
    
    if [ "$build_type" = "all" ] || [ "$build_type" = "release" ]; then
        log_info "Release artifacts created in dist/ directory"
        log_info "Archives and checksums are ready for distribution"
    fi
}

# Run main function with all arguments
main "$@"