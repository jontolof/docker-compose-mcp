# Architecture Guide

## MCP Server Design
The server implements the Model Context Protocol (MCP) to provide Docker Compose functionality to AI assistants through a JSON-RPC 2.0 interface over stdio. This design allows Claude and other AI tools to execute Docker operations without flooding their context with verbose output.

## Core Components

### 1. MCP Layer (`internal/mcp/`)
- Implements JSON-RPC 2.0 protocol using standard library only
- Handles tool registration, execution, and response formatting
- Manages stdio transport with `bufio` and `encoding/json`

### 2. Docker Compose Layer (`internal/compose/`)
- **Client**: Optimized Docker Compose command execution with caching and metrics
- **Command Execution**: Executes Docker Compose commands via `os/exec`
- **Output Processing**: Handles filtering integration and metrics collection
- **Session Integration**: Manages long-running operations with session IDs

### 3. Filtering System (`internal/filter/`)
- **Intelligent Filtering**: Reduces output by 90%+ while preserving errors
- **Framework-Specific**: Go, Jest, pytest, database migration parsing
- **Pattern Matching**: Configurable regex patterns for different output types
- **Fallback Safety**: Raw output preservation when filtering fails

### 4. Optimization Features (Phase 5)
- **Config Cache** (`internal/cache/`): Smart Docker Compose config caching with MD5 integrity
- **Parallel Executor** (`internal/parallel/`): Worker pools for independent operations  
- **Metrics System** (`internal/metrics/`): Real-time filtering performance tracking
- **Tool Framework** (`internal/tools/`): Reusable MCP tool implementations

### 5. Session Management (`internal/session/`)
- **Long-Running Operations**: File watching, log following, monitoring
- **Session Lifecycle**: Start, status check, stop operations with unique IDs
- **Context Management**: Proper cleanup and resource management
- Pattern-based filtering for test results, build output, and logs
- Maintains 90%+ context reduction while preserving critical data

## Key Design Patterns

1. **Clean Architecture**: Separation of concerns between MCP protocol, business logic, and Docker execution
2. **Repository Pattern**: Abstracts Docker command execution from business logic
3. **Service Layer**: Contains all filtering and intelligence logic
4. **DTO Pattern**: Structured data transfer between layers

## Tool Implementation Flow
```
1. MCP Request → 2. Controller → 3. Service → 4. Repository → 5. Docker
                                       ↓
6. MCP Response ← 7. Controller ← 8. Service (filters output)
```

## Output Filtering Strategy

The server implements intelligent filtering to reduce output while maintaining all essential information:

1. **Test Output**: Preserves failures, package summaries, and coverage
2. **Build Output**: Keeps errors, final status, and timing
3. **Logs**: Filters by level (ERROR > WARN > INFO) and patterns
4. **General**: Removes verbose Docker layer details, downloads, and compilation progress