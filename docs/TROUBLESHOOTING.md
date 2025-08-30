# Troubleshooting Guide

This guide helps you resolve common issues with the Docker Compose MCP Server.

## Quick Diagnostics

### Check MCP Server Status
```bash
# Test if the binary works
./docker-compose-mcp --version 2>/dev/null || echo "Binary issue detected"

# Check if Docker is running
docker info >/dev/null 2>&1 || echo "Docker daemon not running"

# Check if Docker Compose is available
docker compose version >/dev/null 2>&1 || echo "Docker Compose not available"
```

### Test Claude Desktop Integration
1. Open Claude Desktop
2. Look for MCP server status in settings/debug panel
3. Try a simple command: "Show me Docker containers"

## Common Issues

### 1. Binary Not Found or Won't Execute

#### Symptoms
- `command not found: docker-compose-mcp`
- `permission denied`

#### Solutions
```bash
# Check if binary exists and is executable
ls -la /usr/local/bin/docker-compose-mcp

# Make executable if needed
chmod +x /usr/local/bin/docker-compose-mcp

# Add to PATH if needed (add to ~/.bashrc or ~/.zshrc)
export PATH="/usr/local/bin:$PATH"

# Verify installation
which docker-compose-mcp
docker-compose-mcp --help 2>/dev/null || echo "Binary test failed"
```

### 2. Docker Daemon Connection Issues

#### Symptoms
- `Cannot connect to the Docker daemon`
- `docker: command not found`

#### Solutions
```bash
# Start Docker Desktop (macOS/Windows)
open -a Docker

# Start Docker daemon (Linux)
sudo systemctl start docker

# Check Docker status
docker info

# Test Docker access
docker ps

# Fix permission issues (Linux)
sudo usermod -aG docker $USER
# Then logout and login again
```

### 3. Claude Desktop Integration Problems

#### Symptoms
- MCP server not appearing in Claude Desktop
- "Failed to connect to MCP server"
- Commands not working in Claude

#### Solutions

**Check Configuration File Location:**
```bash
# macOS
ls -la ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Windows
dir %APPDATA%\Claude\claude_desktop_config.json

# Linux
ls -la ~/.config/Claude/claude_desktop_config.json
```

**Validate Configuration:**
```json
{
  "mcpServers": {
    "docker-compose-mcp": {
      "command": "/usr/local/bin/docker-compose-mcp",
      "args": [],
      "env": {
        "MCP_LOG_LEVEL": "debug"
      }
    }
  }
}
```

**Restart Process:**
1. Quit Claude Desktop completely
2. Wait 5 seconds
3. Restart Claude Desktop
4. Check MCP server status in settings

### 4. Docker Compose File Issues

#### Symptoms
- `docker-compose.yml not found`
- `Invalid compose file`
- Service startup failures

#### Solutions
```bash
# Check if compose file exists
ls -la docker-compose.yml

# Validate compose file syntax
docker compose config

# Check for common issues
docker compose config --quiet || echo "Syntax errors detected"

# Test with minimal compose file
cat > test-compose.yml << 'EOF'
version: '3.8'
services:
  test:
    image: hello-world
EOF

docker compose -f test-compose.yml up --remove-orphans
```

### 5. Service Health Check Failures

#### Symptoms
- Services showing as unhealthy
- Health checks timing out
- Dependency startup issues

#### Solutions
```bash
# Check service logs
docker compose logs [service-name]

# Test health check manually
docker compose exec [service-name] [health-check-command]

# Disable health checks temporarily
# In docker-compose.yml, comment out healthcheck sections

# Increase health check timeouts
healthcheck:
  interval: 60s
  timeout: 30s
  retries: 5
  start_period: 60s
```

### 6. Port Conflicts

#### Symptoms
- `port is already allocated`
- `bind: address already in use`

#### Solutions
```bash
# Find processes using the port
lsof -i :8080  # Replace with your port
netstat -tulpn | grep :8080

# Kill processes using the port
sudo kill -9 $(lsof -t -i:8080)

# Use different ports in docker-compose.yml
ports:
  - "8081:80"  # Changed from 8080:80
```

### 7. Volume Mount Issues

#### Symptoms
- Files not updating in containers
- Permission denied in containers
- Volume mount failures

#### Solutions
```bash
# Check volume mounts
docker compose config | grep -A 5 volumes

# Fix permission issues (Linux/macOS)
chmod -R 755 ./mounted-directory
chown -R $(whoami):$(whoami) ./mounted-directory

# Use absolute paths in volume mounts
volumes:
  - /absolute/path/to/directory:/container/path

# Check Docker Desktop file sharing settings (macOS/Windows)
# Docker Desktop → Settings → Resources → File Sharing
```

### 8. Network Issues

#### Symptoms
- Services can't communicate
- DNS resolution failures
- External network access issues

#### Solutions
```bash
# Check network configuration
docker compose config | grep -A 10 networks

# Test service connectivity
docker compose exec service1 ping service2

# Check Docker networks
docker network ls
docker network inspect [network-name]

# Reset Docker networks
docker compose down
docker system prune -f --networks
docker compose up
```

### 9. Memory and Resource Issues

#### Symptoms
- Container OOM (Out of Memory) kills
- Slow performance
- Build failures due to resources

#### Solutions
```bash
# Check resource usage
docker stats

# Increase Docker Desktop resources
# Docker Desktop → Settings → Resources → Advanced

# Add memory limits to services
services:
  app:
    image: myapp
    mem_limit: 512m
    memswap_limit: 1g

# Clean up unused resources
docker system prune -a --volumes
```

### 10. MCP Server Performance Issues

#### Symptoms
- Slow responses from Claude
- High memory usage
- Timeouts

#### Solutions
```bash
# Enable debug logging
export MCP_LOG_LEVEL=debug

# Reduce cache size
export MCP_CACHE_SIZE=50

# Decrease parallel workers
export MCP_MAX_WORKERS=2

# Increase timeouts
export MCP_COMMAND_TIMEOUT=10m
export MCP_SHUTDOWN_TIMEOUT=60s
```

## Advanced Troubleshooting

### Enable Debug Mode

Set environment variables for detailed logging:
```bash
export MCP_LOG_LEVEL=debug
export MCP_LOG_FORMAT=json
export MCP_ENABLE_DEBUG=true
```

### Check MCP Server Logs

**macOS:**
```bash
tail -f ~/Library/Logs/Claude/mcp-server-docker-compose-mcp.log
```

**Windows:**
```bash
type %APPDATA%\Claude\Logs\mcp-server-docker-compose-mcp.log
```

**Linux:**
```bash
tail -f ~/.local/share/Claude/logs/mcp-server-docker-compose-mcp.log
```

### Manual MCP Server Testing

Test the server directly:
```bash
# Start server manually
docker-compose-mcp

# In another terminal, send test JSON-RPC request
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | docker-compose-mcp
```

### Docker Compose Debugging

```bash
# Verbose output
docker compose --verbose up

# Debug mode
docker compose --log-level DEBUG up

# Check exact commands being executed
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
docker compose up --build
```

### Performance Profiling

```bash
# Monitor resource usage
docker stats --no-stream

# Check disk usage
docker system df

# Profile specific service
docker compose exec service-name top
```

## Error Reference

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `Cannot connect to the Docker daemon` | Docker not running | Start Docker Desktop/daemon |
| `permission denied` | Insufficient permissions | Check file permissions, user groups |
| `port is already allocated` | Port conflict | Change ports or kill conflicting process |
| `no such file or directory` | Missing files | Check paths, file existence |
| `service 'X' failed to build` | Build issues | Check Dockerfile, dependencies |
| `network 'X' not found` | Network issues | Recreate networks, check config |
| `volume 'X' not found` | Volume issues | Create volumes, check paths |
| `health check timeout` | Service startup slow | Increase health check timeout |

### MCP-Specific Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `MCP server not found` | Binary path incorrect | Check claude_desktop_config.json path |
| `Tool 'X' not found` | Tool not registered | Restart MCP server |
| `Session not found` | Session expired | Create new session |
| `Command timeout` | Operation too slow | Increase MCP_COMMAND_TIMEOUT |
| `Invalid parameters` | Wrong parameter format | Check tool schema requirements |

## Getting Help

### Information to Include in Bug Reports

1. **Environment Information:**
   ```bash
   docker --version
   docker compose version
   uname -a
   ./docker-compose-mcp --version
   ```

2. **Configuration:**
   - Claude Desktop config file
   - Environment variables
   - docker-compose.yml (if relevant)

3. **Logs:**
   - MCP server logs
   - Docker Compose logs
   - Claude Desktop console logs

4. **Steps to Reproduce:**
   - Exact commands used
   - Expected vs actual behavior
   - Screenshots if helpful

### Support Resources

- **GitHub Issues**: Report bugs and feature requests
- **Documentation**: Check docs/ directory for guides
- **Examples**: Use examples/ directory for reference
- **Community**: Join discussions and get help

### Self-Help Checklist

Before asking for help, try:

- [ ] Restart Docker Desktop/daemon
- [ ] Restart Claude Desktop
- [ ] Check all file paths are correct
- [ ] Verify environment variables
- [ ] Test with minimal configuration
- [ ] Review recent changes
- [ ] Check system resources (disk, memory)
- [ ] Update to latest versions
- [ ] Try example projects first