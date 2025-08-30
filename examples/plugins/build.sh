#!/bin/bash

# Build script for example plugins
# This script builds the example plugins as shared libraries

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${SCRIPT_DIR}/bin"

echo "Building example plugins..."

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build workflow plugin
echo "Building workflow plugin..."
cd "$SCRIPT_DIR/workflow-plugin"
go build -buildmode=plugin -o "$OUTPUT_DIR/workflow-plugin.so" main.go

# Build monitoring plugin  
echo "Building monitoring plugin..."
cd "$SCRIPT_DIR/monitoring-plugin"
go build -buildmode=plugin -o "$OUTPUT_DIR/monitoring-plugin.so" main.go

# Build integration plugin
echo "Building integration plugin..."
cd "$SCRIPT_DIR/integration-plugin"  
go build -buildmode=plugin -o "$OUTPUT_DIR/integration-plugin.so" main.go

echo "Plugins built successfully in $OUTPUT_DIR/"
echo ""
echo "Available plugins:"
ls -la "$OUTPUT_DIR/"*.so

echo ""
echo "To use these plugins:"
echo "1. Copy the .so files to your plugin directory (e.g., ~/.docker-compose-mcp/plugins/)"
echo "2. Use the plugin_load tool to load them"
echo "3. Use plugin_list to see loaded plugins and their tools"