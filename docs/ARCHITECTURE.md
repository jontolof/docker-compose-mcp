# Architecture Guide

## MCP Server Design
The server implements the Model Context Protocol (MCP) to provide Docker Compose functionality to AI assistants through a JSON-RPC 2.0 interface over stdio. This design allows Claude and other AI tools to execute Docker operations without flooding their context with verbose output.

## Core Components

### 1. MCP Layer (`internal/mcp/`)
- Implements JSON-RPC 2.0 protocol using standard library only
- Handles tool registration, execution, and response formatting
- Manages stdio transport with `bufio` and `encoding/json`

### 2. Docker Compose Layer (`internal/compose/`)
- **Controller**: Handles MCP requests and orchestrates operations
- **Service**: Business logic for filtering and processing Docker output
- **Repository**: Executes Docker Compose commands via `os/exec`
- **DTO**: Request/response models for structured communication

### 3. Filtering System (`internal/filter/`)
- Intelligent output filtering to extract only essential information
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