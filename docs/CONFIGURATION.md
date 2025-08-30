# Configuration Guide

## Environment Variables

### MCP Server Configuration
```bash
MCP_LOG_LEVEL=info              # debug, info, warn, error
MCP_MAX_OUTPUT_LINES=1000       # Maximum lines before truncation
MCP_FILTER_VERBOSITY=normal     # minimal, normal, verbose
```

### Docker Configuration
```bash
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_PROJECT=         # Optional project name
DOCKER_COMPOSE_TIMEOUT=300      # Command timeout in seconds
```

### Filtering Configuration
```bash
FILTER_TEST_OUTPUT=true         # Apply test filtering
FILTER_BUILD_OUTPUT=true        # Apply build filtering
FILTER_KEEP_ERRORS=true         # Always keep error messages
```