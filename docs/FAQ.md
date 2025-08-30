# Frequently Asked Questions (FAQ)

## General Questions

### Q: What is the Docker Compose MCP Server?
**A:** It's a Model Context Protocol (MCP) server that provides AI assistants like Claude with intelligent access to Docker Compose operations. It filters verbose Docker output by 90%+ while preserving all critical information, making it easier for AI to help with container management.

### Q: How does it reduce context by 90%+?
**A:** The server uses intelligent filtering algorithms that:
- Remove verbose layer downloads and build cache information
- Filter repetitive progress indicators
- Keep only essential status changes, errors, and warnings
- Preserve all critical operational information

### Q: Is it safe to use in production?
**A:** Yes, the server includes production-ready features:
- Comprehensive error handling and logging
- Graceful shutdown and cleanup
- Security restrictions on paths and commands
- Timeout management for long operations
- Read-only operations by default (write operations require explicit confirmation)

## Installation Questions

### Q: Which platforms are supported?
**A:** The server supports:
- **macOS**: Intel (x64) and Apple Silicon (ARM64)
- **Linux**: AMD64 and ARM64 architectures
- **Windows**: AMD64 and ARM64 architectures

### Q: Do I need to install Docker separately?
**A:** Yes, you need:
- Docker Desktop (macOS/Windows) or Docker Engine (Linux)
- Docker Compose (usually included with Docker Desktop)
- The docker-compose-mcp binary

### Q: How do I update to a new version?
**A:** Run the installation script again:
```bash
curl -fsSL https://raw.githubusercontent.com/[repo]/main/scripts/install.sh | bash
```
Or download the new binary manually and replace the old one.

## Claude Desktop Integration

### Q: Why don't I see the MCP server in Claude Desktop?
**A:** Check these common issues:
1. Incorrect binary path in claude_desktop_config.json
2. Claude Desktop not restarted after config changes
3. Binary not executable (`chmod +x docker-compose-mcp`)
4. Configuration file in wrong location

### Q: Where is the Claude Desktop configuration file?
**A:** 
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

### Q: Can I use multiple MCP servers with Claude Desktop?
**A:** Yes, add multiple servers to your config:
```json
{
  "mcpServers": {
    "docker-compose-mcp": { ... },
    "other-mcp-server": { ... }
  }
}
```

## Usage Questions

### Q: What Docker Compose commands are supported?
**A:** The server provides 14 tools covering all major operations:
- **Core**: up, down, ps, logs, build
- **Development**: exec, test
- **Monitoring**: watch (start/stop/status)
- **Database**: migrate, db_reset, db_backup
- **Performance**: optimization metrics

### Q: Can I use it with existing Docker Compose projects?
**A:** Yes! Just navigate to any directory with a docker-compose.yml file and ask Claude to help with Docker operations.

### Q: Does it work with Docker Compose v1 (docker-compose) and v2 (docker compose)?
**A:** The server uses standard Docker Compose commands that work with both versions. It automatically detects the available version.

### Q: Can I run multiple watch sessions simultaneously?
**A:** Yes, the server supports multiple concurrent sessions with unique session IDs. You can start, stop, and monitor different watch operations independently.

## Configuration Questions

### Q: How do I change the log level?
**A:** Set the environment variable:
```bash
export MCP_LOG_LEVEL=debug  # Options: debug, info, warn, error
```
Or add to your Claude Desktop config:
```json
{
  "env": {
    "MCP_LOG_LEVEL": "debug"
  }
}
```

### Q: Can I customize the cache size?
**A:** Yes, configure caching options:
```bash
export MCP_CACHE_SIZE=200
export MCP_CACHE_MAX_AGE=1h
export MCP_ENABLE_CACHE=true
```

### Q: How do I increase timeout for long operations?
**A:** Adjust timeout settings:
```bash
export MCP_COMMAND_TIMEOUT=10m
export MCP_SHUTDOWN_TIMEOUT=60s
export MCP_SESSION_TIMEOUT=2h
```

## Performance Questions

### Q: How much memory does the server use?
**A:** Typically 10-50MB depending on:
- Cache size (configurable)
- Number of concurrent operations
- Complexity of Docker Compose projects
- Enabled features (metrics, parallel execution)

### Q: Can I run it on low-resource systems?
**A:** Yes, adjust these settings for lighter usage:
```bash
export MCP_CACHE_SIZE=20
export MCP_MAX_WORKERS=1
export MCP_ENABLE_METRICS=false
export MCP_ENABLE_PARALLEL=false
```

### Q: Does it work with large Docker Compose projects?
**A:** Yes, it's optimized for large projects with:
- Smart configuration caching
- Parallel execution for independent operations
- Streaming output processing
- Resource limits and timeouts

## Security Questions

### Q: What security measures are in place?
**A:** The server includes:
- Path restrictions preventing access to system directories
- Command validation allowing only safe Docker Compose operations
- No direct file system write access
- Timeout protection preventing resource exhaustion
- Safe defaults for all configurations

### Q: Can it access my files outside the project directory?
**A:** No, the server:
- Only operates on docker-compose.yml in the current directory
- Cannot access system directories (/etc, /usr, etc.)
- Cannot execute arbitrary commands
- Only uses Docker Compose for container operations

### Q: Is it safe to run database operations?
**A:** Yes, but with safeguards:
- Database reset requires explicit confirmation
- Automatic backup options before destructive operations
- Extended timeouts for long database operations
- All operations run inside Docker containers

## Troubleshooting Questions

### Q: Commands work in terminal but not through Claude?
**A:** This usually indicates:
- Binary path issues in Claude Desktop config
- Environment variable differences
- Working directory differences
- Permission issues

Try testing the binary directly: `docker-compose-mcp`

### Q: Why are some Docker operations slow?
**A:** Possible causes:
- Container startup time (check health checks)
- Image downloads (first time only)
- Resource constraints (CPU, memory, disk)
- Network connectivity issues

### Q: How do I debug issues?
**A:** Enable debug mode:
```bash
export MCP_LOG_LEVEL=debug
export MCP_ENABLE_DEBUG=true
```
Then check the logs in your system's log directory.

## Advanced Questions

### Q: Can I extend the server with custom tools?
**A:** The server is designed to be comprehensive for Docker Compose operations. For custom tools, you would need to:
1. Fork the repository
2. Add new tools in `internal/tools/`
3. Register them in `cmd/server/main.go`
4. Rebuild the binary

### Q: Can I use it with Docker Swarm or Kubernetes?
**A:** No, this server is specifically designed for Docker Compose. For other orchestration platforms, you would need different MCP servers.

### Q: Does it support Docker multi-stage builds?
**A:** Yes, it works with any Docker Compose configuration including multi-stage builds, multi-platform builds, and complex service dependencies.

### Q: Can I use it in CI/CD pipelines?
**A:** While primarily designed for interactive use with Claude, you could use it in CI/CD by:
- Running the binary directly
- Parsing the filtered JSON output
- Using appropriate timeout configurations

However, for CI/CD, you might prefer direct Docker Compose commands.

## Integration Questions

### Q: Does it work with other AI assistants besides Claude?
**A:** The server implements the standard MCP protocol, so it should work with any MCP-compatible AI assistant. However, it's primarily tested with Claude Desktop.

### Q: Can I integrate it with VS Code or other editors?
**A:** Currently, it's designed for Claude Desktop integration. Editor integration would require:
- MCP client implementation in the editor
- Or a bridge service to translate editor requests to MCP calls

### Q: Does it work with remote Docker hosts?
**A:** Yes, set the DOCKER_HOST environment variable:
```bash
export DOCKER_HOST=tcp://remote-docker-host:2376
```
The server will use your Docker client configuration.

## Development Questions

### Q: How do I contribute to the project?
**A:** 
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Q: How do I report bugs?
**A:** Create an issue on GitHub with:
- Environment information (OS, Docker version, etc.)
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### Q: Can I request new features?
**A:** Yes! Create a feature request issue with:
- Description of the desired functionality
- Use case and benefits
- Possible implementation approach

## Getting More Help

If your question isn't answered here:
1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review the [Integration Guide](CLAUDE_DESKTOP_INTEGRATION.md)
3. Try the example projects in the `examples/` directory
4. Create an issue on GitHub with detailed information