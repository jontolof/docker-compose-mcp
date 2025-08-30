# Development Guide

## Project Status: Phase 5 Complete ✅
The project is in a production-ready state with all 14 MCP tools implemented, optimization features complete, and comprehensive test coverage.

## Initial Setup
```bash
# Project is already initialized - clone and setup:
git clone https://github.com/[your-username]/docker-compose-mcp
cd docker-compose-mcp

# Install dependencies
go mod tidy

# Verify setup
go version  # Should be 1.24.0+
```

## Build and Run Commands
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

## Testing Strategy

Current test status: **13 passing test cases** covering all major functionality.

- **Unit Tests**: Filter algorithms, caching, parallel execution, metrics ✅
- **Integration Tests**: MCP tool registration, optimization features ✅
- **MCP Protocol Tests**: JSON-RPC message handling ✅
- **Coverage Achieved**: Comprehensive coverage of core functionality

## Code Standards

- **Go Version**: 1.24.0+
- **Style**: Standard Go formatting (`go fmt`)
- **Dependencies**: Minimal - prefer standard library
- **Error Handling**: Always wrap errors with context
- **Interfaces**: Define for all external dependencies
- **Documentation**: Document all exported functions