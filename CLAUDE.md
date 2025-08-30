# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT: Communication Guidelines

**NEVER reference Claude, Claude Code, Anthropic, or any AI assistant in:**
- Code comments
- Git commit messages  
- Documentation
- Error messages
- Log output
- Variable/function names
- Any project communication

This is a professional open-source project. All code and documentation should be written as if by a human developer, focusing solely on the technical implementation without mentioning AI assistance.

## Project Overview

A Model Context Protocol (MCP) server for Docker Compose operations that filters verbose Docker output to essential information only, preventing AI assistant context flooding.

**Core Mission**: Reduce Docker Compose command output by 90%+ while maintaining complete operational capability.

## Development Commands

### Initial Setup
```bash
# Initialize Go module (first time only)
go mod init github.com/[your-username]/docker-compose-mcp

# Install dependencies
go mod tidy

# Verify setup
go version  # Should be 1.24.0+
```

### Build and Run
```bash
# Build the MCP server
go build -o docker-compose-mcp cmd/server/main.go

# Run tests
go test ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test package
go test -v ./internal/compose/service

# Run with race detection
go test -race ./...

# Format code
go fmt ./...

# Lint code (install golangci-lint first)
golangci-lint run

# Build Docker image
docker build -t docker-compose-mcp:latest .

# Run server in development mode
MCP_LOG_LEVEL=debug go run cmd/server/main.go stdio
```

## High-Level Architecture

### MCP Server Design
The server implements the Model Context Protocol (MCP) to provide Docker Compose functionality to AI assistants through a JSON-RPC 2.0 interface over stdio. This design allows Claude and other AI tools to execute Docker operations without flooding their context with verbose output.

### Core Components

1. **MCP Layer** (`internal/mcp/`)
   - Implements JSON-RPC 2.0 protocol using standard library only
   - Handles tool registration, execution, and response formatting
   - Manages stdio transport with `bufio` and `encoding/json`

2. **Docker Compose Layer** (`internal/compose/`)
   - **Controller**: Handles MCP requests and orchestrates operations
   - **Service**: Business logic for filtering and processing Docker output
   - **Repository**: Executes Docker Compose commands via `os/exec`
   - **DTO**: Request/response models for structured communication

3. **Filtering System** (`internal/filter/`)
   - Intelligent output filtering to extract only essential information
   - Pattern-based filtering for test results, build output, and logs
   - Maintains 90%+ context reduction while preserving critical data

### Key Design Patterns

1. **Clean Architecture**: Separation of concerns between MCP protocol, business logic, and Docker execution
2. **Repository Pattern**: Abstracts Docker command execution from business logic
3. **Service Layer**: Contains all filtering and intelligence logic
4. **DTO Pattern**: Structured data transfer between layers

### Tool Implementation Flow
```
1. MCP Request → 2. Controller → 3. Service → 4. Repository → 5. Docker
                                       ↓
6. MCP Response ← 7. Controller ← 8. Service (filters output)
```

## Available MCP Tools

### Essential Tools
- `docker_compose_build`: Build with filtered output
- `docker_compose_test`: Run tests with smart filtering
- `docker_compose_coverage`: Generate coverage reports
- `docker_compose_up`: Start services with health monitoring
- `docker_compose_down`: Stop and clean up containers
- `docker_compose_logs`: Get filtered service logs
- `docker_compose_exec`: Execute commands in containers

### Database Tools
- `docker_compose_migrate`: Run database migrations
- `docker_compose_db_reset`: Reset database to clean state

### Advanced Tools
- `docker_compose_watch`: Watch files and rebuild
- `docker_compose_benchmark`: Run performance benchmarks

## Output Filtering Strategy

The server implements intelligent filtering to reduce output while maintaining all essential information:

1. **Test Output**: Preserves failures, package summaries, and coverage
2. **Build Output**: Keeps errors, final status, and timing
3. **Logs**: Filters by level (ERROR > WARN > INFO) and patterns
4. **General**: Removes verbose Docker layer details, downloads, and compilation progress

## Configuration

Environment variables for runtime configuration:
```bash
# MCP Server
MCP_LOG_LEVEL=info              # debug, info, warn, error
MCP_MAX_OUTPUT_LINES=1000       # Maximum lines before truncation
MCP_FILTER_VERBOSITY=normal     # minimal, normal, verbose

# Docker
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_PROJECT=         # Optional project name
DOCKER_COMPOSE_TIMEOUT=300      # Command timeout in seconds

# Filtering
FILTER_TEST_OUTPUT=true         # Apply test filtering
FILTER_BUILD_OUTPUT=true        # Apply build filtering
FILTER_KEEP_ERRORS=true         # Always keep error messages
```

## Implementation Status

### Current Phase: Planning & Specification
- [x] Project specification (CLAUDE.md)
- [x] Implementation guide (MCP_SERVER_IMPLEMENTATION_GUIDE.md)
- [ ] Go module initialization
- [ ] Basic project structure setup

### Next Steps
1. Initialize Go module: `go mod init github.com/[your-username]/docker-compose-mcp`
2. Create project structure as defined
3. Implement MCP server scaffold with stdio transport
4. Add Docker Compose command execution
5. Implement intelligent filtering
6. Test with Claude Desktop

## Testing Strategy

- **Unit Tests**: Test filtering logic with mocked Docker output
- **Integration Tests**: Test with real Docker commands
- **MCP Protocol Tests**: Validate JSON-RPC message handling
- **Coverage Target**: 80%+ code coverage

## Code Standards

- **Go Version**: 1.24.0+
- **Style**: Standard Go formatting (`go fmt`)
- **Dependencies**: Minimal - prefer standard library
- **Error Handling**: Always wrap errors with context
- **Interfaces**: Define for all external dependencies
- **Documentation**: Document all exported functions