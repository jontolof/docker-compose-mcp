# Contributing to docker-compose-mcp

Thank you for your interest in contributing to docker-compose-mcp! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Issues

1. Check existing issues to avoid duplicates
2. Use issue templates when available
3. Provide clear reproduction steps
4. Include relevant system information (OS, Go version, Docker version)

### Suggesting Features

1. Open a discussion in GitHub Discussions first
2. Describe the use case and benefits
3. Consider implementation complexity
4. Wait for maintainer feedback before starting work

### Submitting Code

#### 1. Fork and Clone

```bash
# Fork via GitHub UI, then:
git clone https://github.com/YOUR-USERNAME/docker-compose-mcp.git
cd docker-compose-mcp
git remote add upstream https://github.com/jonttolof/docker-compose-mcp.git
```

#### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

#### 3. Development Setup

```bash
# Initialize Go module
go mod download

# Run tests
go test ./...

# Build locally
go build -o docker-compose-mcp cmd/server/main.go
```

#### 4. Make Your Changes

- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed
- Keep commits focused and atomic

#### 5. Testing Requirements

Before submitting:

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Check formatting
go fmt ./...

# Run linting (if golangci-lint installed)
golangci-lint run
```

#### 6. Commit Guidelines

Follow conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or fixes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

Examples:
```
feat(filter): add pytest output filtering
fix(compose): handle multi-file compose configs
docs(readme): update installation instructions
```

#### 7. Submit Pull Request

1. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Open PR via GitHub with:
   - Clear title describing the change
   - Description of what and why
   - Reference any related issues
   - Screenshots/examples if applicable

3. PR Requirements:
   - All tests must pass
   - Code must be formatted (`go fmt`)
   - Maintain or increase test coverage
   - Update documentation if needed

### Pull Request Review Process

1. **Automated Checks**: CI will run tests and linting
2. **Code Review**: Maintainer will review within 2-5 days
3. **Feedback**: Address any requested changes
4. **Merge**: Only maintainer can merge to master

## Development Guidelines

### Code Standards

#### Go Best Practices
- Use Go 1.24.0+ features appropriately
- Prefer standard library over external dependencies
- Handle errors explicitly
- Use interfaces for testability
- Document exported functions

#### Project-Specific Rules
- Filtering must preserve all errors and warnings
- Output reduction should maintain information integrity
- MCP protocol compliance is mandatory
- Tests required for new filtering patterns

### Architecture Guidelines

Follow the established clean architecture:

```
internal/
├── mcp/        # Protocol layer - handles JSON-RPC
├── compose/    # Business layer - Docker Compose logic
├── filter/     # Core feature - output filtering
└── session/    # State management
```

### Testing Standards

#### Unit Tests
- Mock external dependencies
- Test edge cases
- Aim for 80%+ coverage

#### Integration Tests
- Test with real Docker commands
- Verify filtering accuracy
- Check MCP protocol compliance

Example test:
```go
func TestBuildOutputFilter(t *testing.T) {
    input := loadFixture(t, "docker-build-verbose.txt")
    output := filter.FilterBuildOutput(input)
    
    assert.Less(t, len(output), len(input)/10)
    assert.Contains(t, output, "ERROR")
    assert.NotContains(t, output, "Downloading")
}
```

## Filtering Contribution Guidelines

When adding new filtering patterns:

1. **Identify Pattern**: Document the verbose output pattern
2. **Design Filter**: Create regex or parsing logic
3. **Preserve Critical Info**: Never filter errors/warnings
4. **Add Tests**: Include real-world samples
5. **Measure Reduction**: Verify 90%+ reduction goal

Example:
```go
// internal/filter/patterns.go
var TestOutputPatterns = []Pattern{
    {
        Name: "go_test_pass",
        Regex: regexp.MustCompile(`^ok\s+\S+\s+\S+s$`),
        Keep: true,  // Keep package summaries
    },
    // Add your pattern here
}
```

## Release Process

1. Maintainer creates release branch
2. Version bump and changelog update
3. Testing and review
4. Tag and release via GitHub
5. Binary distribution updates

## Getting Help

- **Documentation**: Check README.md and docs/
- **Discussions**: Use GitHub Discussions for questions
- **Issues**: Report bugs via GitHub Issues
- **Contact**: Reach out to maintainer via GitHub

## Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Mentioned in release notes
- Credited in relevant documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for helping make docker-compose-mcp better!