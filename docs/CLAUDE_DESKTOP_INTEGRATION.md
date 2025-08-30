# Claude Desktop Integration Guide

This guide explains how to integrate docker-compose-mcp with Claude Desktop.

## Prerequisites

1. **Claude Desktop installed** - Download from [Claude Desktop](https://claude.ai/download)
2. **docker-compose-mcp binary** - Build from source or download release
3. **Docker and Docker Compose** - Ensure both are installed and running

## Installation Steps

### 1. Build or Download Binary

#### Option A: Build from Source
```bash
# Clone the repository
git clone https://github.com/[username]/docker-compose-mcp.git
cd docker-compose-mcp

# Build the binary
go build -o docker-compose-mcp cmd/server/main.go

# Make it executable and move to PATH
chmod +x docker-compose-mcp
sudo mv docker-compose-mcp /usr/local/bin/
```

#### Option B: Download Release Binary
```bash
# Download latest release (replace with actual URL)
curl -L https://github.com/[username]/docker-compose-mcp/releases/latest/download/docker-compose-mcp-darwin-amd64 -o docker-compose-mcp

# Make executable and move to PATH
chmod +x docker-compose-mcp
sudo mv docker-compose-mcp /usr/local/bin/
```

### 2. Configure Claude Desktop

#### Find Configuration Directory

**macOS:**
```bash
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows:**
```bash
%APPDATA%\Claude\claude_desktop_config.json
```

**Linux:**
```bash
~/.config/Claude/claude_desktop_config.json
```

#### Add MCP Server Configuration

Edit your `claude_desktop_config.json` file:

```json
{
  "mcpServers": {
    "docker-compose-mcp": {
      "command": "/usr/local/bin/docker-compose-mcp",
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
```

### 3. Restart Claude Desktop

After updating the configuration:
1. Quit Claude Desktop completely
2. Restart the application
3. Verify the MCP server appears in the status/debug panel

## Available Tools

Once integrated, you'll have access to these Docker Compose tools:

### Core Operations
- `compose_up` - Start services with intelligent filtering
- `compose_down` - Stop and clean up services
- `compose_ps` - List running containers
- `compose_logs` - View filtered logs
- `compose_build` - Build services with progress filtering

### Testing & Development
- `compose_exec` - Execute commands in containers
- `compose_test` - Run tests with framework-specific parsing

### Session-Based Monitoring
- `compose_watch_start` - Start file watching with auto-rebuild
- `compose_watch_stop` - Stop watch sessions
- `compose_watch_status` - Get watch session status

### Database Operations
- `compose_migrate` - Run database migrations
- `compose_db_reset` - Reset database with safety confirmations
- `compose_db_backup` - Create, restore, and list backups

### Performance & Optimization
- `compose_optimization` - Get metrics and performance data

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |
| `MCP_LOG_FORMAT` | `text` | Log format: text, json |
| `MCP_WORK_DIR` | `.` | Working directory for operations |
| `MCP_ENABLE_CACHE` | `true` | Enable configuration caching |
| `MCP_CACHE_SIZE` | `100` | Maximum cache entries |
| `MCP_CACHE_MAX_AGE` | `30m` | Cache entry maximum age |
| `MCP_ENABLE_METRICS` | `true` | Enable performance metrics |
| `MCP_ENABLE_PARALLEL` | `true` | Enable parallel execution |
| `MCP_MAX_WORKERS` | `4` | Maximum parallel workers |
| `MCP_COMMAND_TIMEOUT` | `5m` | Command execution timeout |
| `MCP_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `MCP_MAX_SESSIONS` | `10` | Maximum concurrent sessions |
| `COMPOSE_FILE` | `docker-compose.yml` | Compose file path |
| `COMPOSE_PROJECT_NAME` | auto-detected | Project name override |

### Advanced Configuration

For production environments:

```json
{
  "mcpServers": {
    "docker-compose-mcp": {
      "command": "/usr/local/bin/docker-compose-mcp",
      "args": [],
      "env": {
        "MCP_LOG_LEVEL": "warn",
        "MCP_LOG_FORMAT": "json",
        "MCP_LOG_FILE": "/var/log/docker-compose-mcp.log",
        "MCP_ENABLE_CACHE": "true",
        "MCP_CACHE_SIZE": "200",
        "MCP_CACHE_MAX_AGE": "1h",
        "MCP_ENABLE_METRICS": "true",
        "MCP_ENABLE_PARALLEL": "true",
        "MCP_MAX_WORKERS": "8",
        "MCP_COMMAND_TIMEOUT": "10m",
        "MCP_SESSION_TIMEOUT": "2h",
        "MCP_MAX_SESSIONS": "20"
      }
    }
  }
}
```

## Usage Examples

### Starting Services
Ask Claude: "Start my Docker services in detached mode"
- Uses: `compose_up` with `detach: true`
- Output: Filtered startup logs showing only essential information

### Running Tests
Ask Claude: "Run the tests in my web service with coverage"
- Uses: `compose_test` with `service: "web"` and `coverage: true`
- Output: Parsed test results with coverage information

### Database Migration
Ask Claude: "Run database migrations up to version 123"
- Uses: `compose_migrate` with `direction: "up"` and `target: "123"`
- Output: Migration progress without verbose SQL output

### File Watching
Ask Claude: "Start watching for file changes and rebuild when needed"
- Uses: `compose_watch_start` with `build: true`
- Creates session for continuous monitoring

## Troubleshooting

### Common Issues

#### 1. Binary Not Found
**Error:** `command not found: docker-compose-mcp`
**Solution:** Verify binary is in PATH or use absolute path in config

#### 2. Permission Denied
**Error:** `permission denied`
**Solution:** Ensure binary has execute permissions: `chmod +x docker-compose-mcp`

#### 3. Docker Daemon Not Running
**Error:** `Cannot connect to the Docker daemon`
**Solution:** Start Docker Desktop or Docker daemon

#### 4. Compose File Not Found
**Error:** `docker-compose.yml not found`
**Solution:** Set `COMPOSE_FILE` environment variable to correct path

#### 5. Timeout Issues
**Error:** `operation timed out`
**Solution:** Increase `MCP_COMMAND_TIMEOUT` for long-running operations

### Debug Mode

Enable debug logging:
```json
{
  "env": {
    "MCP_LOG_LEVEL": "debug",
    "MCP_ENABLE_DEBUG": "true"
  }
}
```

### Log Files

Check logs in:
- **macOS:** `~/Library/Logs/Claude/mcp-server-docker-compose-mcp.log`
- **Windows:** `%APPDATA%\Claude\Logs\mcp-server-docker-compose-mcp.log`
- **Linux:** `~/.local/share/Claude/logs/mcp-server-docker-compose-mcp.log`

## Performance Optimization

### For Large Projects
```json
{
  "env": {
    "MCP_CACHE_SIZE": "500",
    "MCP_CACHE_MAX_AGE": "2h",
    "MCP_MAX_WORKERS": "8",
    "MCP_COMMAND_TIMEOUT": "15m"
  }
}
```

### For Development
```json
{
  "env": {
    "MCP_LOG_LEVEL": "debug",
    "MCP_ENABLE_METRICS": "true",
    "MCP_ENABLE_PARALLEL": "true"
  }
}
```

### For Production
```json
{
  "env": {
    "MCP_LOG_LEVEL": "warn",
    "MCP_LOG_FORMAT": "json",
    "MCP_ENABLE_CACHE": "true",
    "MCP_ENABLE_METRICS": "false"
  }
}
```

## Security Considerations

1. **Path Restrictions:** Binary automatically restricts access to system directories
2. **Command Validation:** Only allowed Docker Compose commands are permitted
3. **Timeout Protection:** All operations have configurable timeouts
4. **Safe Defaults:** Conservative defaults prevent resource exhaustion

## Integration Testing

Test the integration:

1. **Basic connectivity:**
   Ask Claude: "List my running Docker containers"

2. **Complex operations:**
   Ask Claude: "Start my services, run tests, and show me the results"

3. **Session management:**
   Ask Claude: "Start watching my project for changes"

4. **Error handling:**
   Try operations without Docker running to test error messages

## Support

For issues:
1. Check logs for detailed error information
2. Verify Docker and Compose are working independently
3. Test MCP server manually: `docker-compose-mcp`
4. Report issues with logs and configuration details