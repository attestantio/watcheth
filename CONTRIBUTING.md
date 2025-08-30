# Contributing

## Maintainers

- [Hoanh An](https://github.com/hoanhan101)
- [Chris Berry](https://github.com/Bez625)

## Development Setup

**Prerequisites:**

- Go 1.21+
- golangci-lint
- Make (optional)

**Project Structure:**

```
watcheth/
├── cmd/           # CLI commands
├── internal/      # Core logic
│   ├── consensus/ # Beacon clients
│   ├── execution/ # Execution clients
│   ├── validator/ # Vouch support
│   └── monitor/   # UI display
└── docs/          # Documentation
```

## Building

```bash
# Quick build
go build

# Using Make
make build       # Current platform
make build-all   # All platforms
make install     # Install to GOPATH/bin

# Cross-compile
GOOS=linux GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build
```

## Testing

```bash
# Run tests
go test -race ./...
make test

# With coverage
go test -race -cover ./...

# Specific test
go test -v -run TestName ./...
```

## Code Quality

```bash
# Format
go fmt ./...

# Lint
golangci-lint run ./...
golangci-lint run --fix ./...

# Dependencies
go mod tidy
```

## Pull Request Process

1. **Before submitting:**

   ```bash
   go fmt ./...
   golangci-lint run ./...
   go test -race ./...
   ```

2. **Commit format:**

   ```
   type: description

   feat: add new feature
   fix: resolve bug
   docs: update documentation
   test: add tests
   refactor: improve code
   ```

3. **PR must have:**
   - Clear title and description
   - Passing CI checks
   - Updated tests/docs if needed

## Release Process

See [docs/workflows.md](docs/workflows.md) for CI/CD details.

```bash
# Create release
git tag v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Getting Help

- [Issues](https://github.com/attestantio/watcheth/issues)
- [Documentation](docs/)
