# GitHub Actions Workflows

This document describes the CI/CD workflows configured for watcheth.

## Overview

watcheth uses GitHub Actions for continuous integration and deployment with three main workflows:

1. **Linting** - Code quality checks using golangci-lint
2. **Testing** - Automated test execution
3. **Release** - Multi-platform binary builds and distribution

## Workflows

### Linting (`golangci-lint.yml`)

**Trigger:** Push to main/master branches or any pull request

**Purpose:** Ensures code quality and consistency across the codebase

**Configuration:**
- Uses golangci-lint with extensive checks enabled
- Configuration defined in `.golangci.yml`
- Only reports new issues in pull requests
- 60-minute timeout for large codebases

**Local Execution:**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run ./...
```

### Testing (`test.yml`)

**Trigger:** Push to main/master branches or any pull request

**Purpose:** Runs all unit and integration tests

**Configuration:**
- Executes with race detection enabled (`-race`)
- 30-minute timeout
- Uses Go 1.23

**Local Execution:**
```bash
# Run all tests
go test -race -timeout=30m ./...

# Run tests with coverage
go test -race -cover ./...

# Run tests with verbose output
go test -race -v ./...
```

### Release (`release.yml`)

**Trigger:** Push of version tags (v* or t*)

**Purpose:** Builds and releases binaries for multiple platforms

**Supported Platforms:**
- Linux AMD64 (x86_64)
- Linux ARM64 (aarch64) 
- macOS AMD64 (Intel)
- macOS ARM64 (Apple Silicon)
- Windows AMD64

**Release Process:**

1. **Tag the release:**
```bash
# Create a version tag
git tag v1.0.0 -m "Release version 1.0.0"

# Push the tag to trigger the workflow
git push origin v1.0.0
```

2. **Workflow Actions:**
   - Creates a draft GitHub release
   - Builds binaries for all platforms
   - Generates SHA256 checksums
   - Uploads artifacts to the release

3. **Release Artifacts:**
   Each platform receives:
   - Compressed binary (`.tar.gz` for Unix, `.zip` for Windows)
   - SHA256 checksum file

4. **Version Injection:**
   The version is automatically injected into the binary at build time via:
   ```
   -ldflags="-X github.com/watcheth/watcheth/internal/version.Version=1.0.0"
   ```

## Development Workflow

### Before Committing

1. **Format your code:**
```bash
go fmt ./...
```

2. **Run linting locally:**
```bash
golangci-lint run ./...
```

3. **Run tests:**
```bash
go test -race ./...
```

### Pull Request Process

1. Create a feature branch
2. Make your changes
3. Push to your fork/branch
4. Open a pull request
5. Wait for CI checks to pass:
   - ✅ Linting must pass
   - ✅ Tests must pass
6. Address any feedback
7. Merge when approved

### Creating a Release

1. **Ensure main branch is ready:**
```bash
git checkout main
git pull origin main
```

2. **Update version if needed:**
   - Update any version constants
   - Update CHANGELOG.md
   - Commit changes

3. **Create and push tag:**
```bash
# Semantic versioning: vMAJOR.MINOR.PATCH
git tag v1.2.3 -m "Release v1.2.3: Brief description"
git push origin v1.2.3
```

4. **Monitor the release:**
   - Go to Actions tab on GitHub
   - Watch the release workflow
   - Once complete, go to Releases page
   - Edit the draft release
   - Add release notes
   - Publish the release

### Testing Releases

For testing the release process without creating an official release:
```bash
# Use 't' prefix for test releases
git tag t1.2.3-beta1
git push origin t1.2.3-beta1
```

## Troubleshooting

### Linting Failures

If linting fails, check the specific issues:
```bash
# See all issues
golangci-lint run ./...

# Auto-fix some issues
golangci-lint run --fix ./...

# Check specific file
golangci-lint run path/to/file.go
```

### Test Failures

For test failures:
```bash
# Run specific test
go test -v -run TestName ./path/to/package

# Run with more detail
go test -race -v ./...

# Check for race conditions
go test -race ./...
```

### Release Build Issues

For release problems:
1. Check Go version compatibility (requires 1.23+)
2. Ensure all dependencies are vendored or available
3. Test build locally:
```bash
# Test Linux build
GOOS=linux GOARCH=amd64 go build -o watcheth

# Test Windows build  
GOOS=windows GOARCH=amd64 go build -o watcheth.exe

# Test macOS build
GOOS=darwin GOARCH=amd64 go build -o watcheth
```

## Configuration Files

### `.golangci.yml`

Controls linting rules and settings:
- Enabled linters list
- Linter-specific configurations
- Issue exclusion rules
- Timeout and performance settings

### `.github/workflows/`

Contains workflow definitions:
- `golangci-lint.yml` - Linting workflow
- `test.yml` - Testing workflow  
- `release.yml` - Release build workflow

## Best Practices

1. **Keep workflows fast:** Optimize for quick feedback
2. **Test locally first:** Run linting and tests before pushing
3. **Use semantic versioning:** Follow vMAJOR.MINOR.PATCH format
4. **Document releases:** Add comprehensive release notes
5. **Monitor CI status:** Fix failures promptly
6. **Cache dependencies:** Workflows use caching for speed

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Go Testing Documentation](https://golang.org/doc/tutorial/add-a-test)
- [Semantic Versioning](https://semver.org/)