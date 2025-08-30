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

## Quick Start

```bash
# Initialize Go module
go mod init github.com/[your-username]/docker-compose-mcp

# Build and run
go build -o docker-compose-mcp cmd/server/main.go
MCP_LOG_LEVEL=debug go run cmd/server/main.go stdio

# Run tests
go test ./...
```

## Documentation

- **[Development Guide](docs/DEVELOPMENT.md)**: Build commands, testing, code standards
- **[Architecture Guide](docs/ARCHITECTURE.md)**: System design, components, patterns
- **[MCP Tools Reference](docs/MCP_TOOLS.md)**: Available tools and their usage
- **[Configuration Guide](docs/CONFIGURATION.md)**: Environment variables and settings

## Project Structure

```
docker-compose-mcp/
├── cmd/server/          # Server entry point
├── internal/
│   ├── mcp/            # MCP protocol implementation
│   ├── compose/        # Docker Compose logic
│   └── filter/         # Output filtering system
├── docs/               # Detailed documentation
└── tests/              # Test files
```

## Implementation Status

### Current Phase: Planning & Specification
- [x] Project specification (CLAUDE.md)
- [x] Implementation guide (MCP_SERVER_IMPLEMENTATION_GUIDE.md)
- [x] Documentation structure (docs/)
- [ ] Go module initialization
- [ ] Basic project structure setup

### Next Steps
1. Initialize Go module with proper path
2. Create project structure as defined
3. Implement MCP server scaffold with stdio transport
4. Add Docker Compose command execution
5. Implement intelligent filtering
6. Test with Claude Desktop

## Key Features

- **Intelligent Filtering**: Reduces Docker output by 90%+ while preserving essential information
- **MCP Protocol**: JSON-RPC 2.0 over stdio for AI assistant integration
- **Clean Architecture**: Separated concerns with repository pattern and service layers
- **Minimal Dependencies**: Primarily uses Go standard library

## Development Workflow

1. Check [Development Guide](docs/DEVELOPMENT.md) for commands
2. Review [Architecture Guide](docs/ARCHITECTURE.md) before major changes
3. Use [MCP Tools Reference](docs/MCP_TOOLS.md) for tool implementation
4. Configure via [Configuration Guide](docs/CONFIGURATION.md)