# CI/CD Workflows

## GitHub Actions

### 1. Linting (`golangci-lint.yml`)

- **Trigger:** Push to main or PR
- **Runs:** `golangci-lint run ./...`
- **Config:** `.golangci.yml`

### 2. Testing (`test.yml`)

- **Trigger:** Push to main or PR
- **Runs:** `go test -race ./...`
- **Go version:** 1.23

### 3. Release (`release.yml`)

- **Trigger:** Version tags (`v*` or `t*`)
- **Builds:** Linux, macOS (AMD64 & ARM64)
- **Creates:** Draft release with binaries

## Development Workflow

### Before Committing

```bash
go fmt ./...
golangci-lint run ./...
go test -race ./...
```

### Creating a Release

```bash
# Tag and push
git tag v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Test releases use 't' prefix
git tag t1.0.0-beta
git push origin t1.0.0-beta
```

## Local Testing

### Linting

```bash
# Install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run
golangci-lint run ./...
golangci-lint run --fix ./...  # Auto-fix
```

### Tests

```bash
go test -race ./...             # All tests
go test -v -run TestName ./...  # Specific test
go test -race -cover ./...      # With coverage
```

### Build Testing

```bash
# Cross-compile test
GOOS=linux GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build
```

## Troubleshooting

### Failed CI?

1. Run locally first: `make test && make lint`
2. Check Go version: requires 1.23+
3. Review workflow logs in Actions tab

### Release Issues?

1. Ensure tag format: `v1.2.3`
2. Check all platforms build locally
3. Verify dependencies: `go mod tidy`
