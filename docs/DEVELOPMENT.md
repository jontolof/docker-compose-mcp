# Development Guide

## Initial Setup
```bash
# Initialize Go module (first time only)
go mod init github.com/[your-username]/docker-compose-mcp

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