# Configuration Guide

## MCP Server Configuration

The Docker Compose MCP server supports extensive configuration through client options and environment variables.

### Client Options (Programmatic)
```go
opts := &compose.ClientOptions{
    WorkDir:         ".",                    // Working directory
    EnableCache:     true,                   // Smart config caching
    EnableMetrics:   true,                   // Performance metrics
    EnableParallel:  true,                   // Parallel execution
    CacheSize:       100,                    // Max cache entries
    CacheMaxAge:     30 * time.Minute,      // Cache TTL
    MaxWorkers:      4,                      // Parallel workers
    CommandTimeout:  5 * time.Minute,       // Command timeout
}
```

### Environment Variables

#### Core MCP Configuration
```bash
MCP_LOG_LEVEL=info              # debug, info, warn, error
MCP_MAX_OUTPUT_LINES=1000       # Maximum lines before truncation
MCP_FILTER_VERBOSITY=normal     # minimal, normal, verbose
```

#### Optimization Features (Phase 5)
```bash
# Caching
COMPOSE_CACHE_ENABLED=true      # Enable config caching
COMPOSE_CACHE_SIZE=100          # Max cache entries
COMPOSE_CACHE_TTL=1800          # Cache TTL in seconds

# Parallel Execution  
COMPOSE_PARALLEL_ENABLED=true   # Enable parallel execution
COMPOSE_MAX_WORKERS=4           # Worker pool size

# Metrics
COMPOSE_METRICS_ENABLED=true    # Enable performance metrics
COMPOSE_METRICS_EXPORT=false    # Auto-export metrics
```

#### Docker Configuration
```bash
DOCKER_COMPOSE_FILE=docker-compose.yml  # Compose file path
DOCKER_COMPOSE_PROJECT=                  # Optional project name
DOCKER_COMPOSE_TIMEOUT=300               # Command timeout in seconds
```

#### Filtering Configuration
```bash
FILTER_TEST_OUTPUT=true         # Apply test framework filtering
FILTER_BUILD_OUTPUT=true        # Apply build output filtering
FILTER_MIGRATION_OUTPUT=true    # Apply database migration filtering
FILTER_KEEP_ERRORS=true         # Always preserve error messages
FILTER_REDUCTION_TARGET=0.9     # Target 90% output reduction
```