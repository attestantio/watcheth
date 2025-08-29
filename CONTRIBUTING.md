# Contributing to watcheth

## Maintainers

The maintainers for watcheth are:

  - [Hoanh An](https://github.com/hoanhan101)
  - [Chris Berry](https://github.com/Bez625)

## Issues

If you have found a problem with watcheth or would like to suggest a feature, please check the [open issues](https://github.com/attestantio/watcheth/issues) to see if it has already been raised.  If not, feel free to open a new issue and provide as much detail as possible about the issue or feature.

## Development

### Prerequisites

- Go 1.23 or later
- Make (optional, for using Makefile targets)
- golangci-lint (for running linters locally)

### Building

```bash
# Build for current platform
make build

# Build for Linux AMD64 (common for servers)
make build-linux

# Build for all major platforms
make build-all

# Manual build without Make
go build -o watcheth

# Build with version info
go build -ldflags "-X github.com/watcheth/watcheth/internal/version.Version=$(git describe --tags --always --dirty)"
```

### Running

```bash
# Run from build directory
./build/watcheth

# Run with custom config
./build/watcheth --config /path/to/watcheth.yml

# Run in debug mode
./build/watcheth --debug

# List nodes once (non-interactive)
./build/watcheth list

# Monitor continuously (interactive)
./build/watcheth monitor
```

## Tests and Linting

### Tests

Tests can be run locally with:

```bash
go test ./...
```

or using Make:

```bash
make test
```

### Linting

Linters can be run locally with:

```bash
golangci-lint run
```

### Code Quality

Additional code quality checks:

```bash
# Format code
go fmt ./...

# Vet code for common issues
go vet ./...

# Update dependencies
go mod tidy
```

## Pull Requests

Please ensure that:
1. All tests pass
2. The linter reports no issues
3. Your code follows the existing code style
4. Commit messages are clear and descriptive