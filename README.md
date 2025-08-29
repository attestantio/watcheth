# watcheth

Unified monitoring for Ethereum consensus, execution, and validator clients.

## Overview

watcheth provides a unified terminal dashboard for monitoring Ethereum infrastructure in real-time. It eliminates the need to juggle multiple endpoints and tools by displaying key metrics from consensus clients, execution clients, and validator clients in a single view with automatic refresh and live log streaming.

## Features

- **Multi-client Support**: Monitor consensus, execution, and validator clients simultaneously
  - Consensus: Prysm, Lighthouse, Teku, Nimbus
  - Execution: Geth, Nethermind, Besu, Erigon
  - Validator: Vouch
- **Real-time Updates**: Automatic refresh with configurable intervals
- **Live Log Viewer**: Real-time log streaming with 100ms refresh rate
- **Clean Terminal UI**: Fixed-height sections with consistent formatting
- **Validator Monitoring**: Track attestations, proposals, and sync committees for Vouch
- **Flexible Configuration**: YAML-based configuration for all node endpoints

## Installation

```bash
go install github.com/attestantio/watcheth@latest
```

Or build from source:

```bash
git clone https://github.com/attestantio/watcheth
cd watcheth
go build
```

## Usage

### Quick Start

Run with default configuration (interactive mode):

```bash
watcheth
# or
watcheth monitor
```

List nodes once (non-interactive):

```bash
watcheth list
```

### Debug Mode

Test API endpoints on consensus, execution, or validator clients:

```bash
# Debug consensus client
watcheth debug http://localhost:5052

# Debug execution client  
watcheth debug http://localhost:8545 --type execution

# Debug Vouch validator client
watcheth debug http://localhost:8081 --type vouch

# Save debug output to file
watcheth debug http://localhost:5052 --output debug-results.txt
```

The debug command tests API endpoints and displays their responses. Use `--output` or `-o` to save results to a file.

### Custom Configuration

Create a `watcheth.yml` file:

```yaml
clients:
  # Consensus clients
  - name: "Prysm Node 1"
    type: consensus
    endpoint: "http://localhost:3500"
  - name: "Lighthouse Node 1"
    type: consensus
    endpoint: "http://localhost:5052"
    
  # Execution clients
  - name: "Geth"
    type: execution
    endpoint: "http://localhost:8545"
    
  # Validator clients
  - name: "Vouch"
    type: vouch
    endpoint: "http://localhost:8081"  # Prometheus metrics endpoint

refresh_interval: 2s
```

Run with custom configuration:

```bash
watcheth --config /path/to/watcheth.yml
```

### Keyboard Shortcuts

- `q` - Quit the application
- `r` - Force refresh all data
- `v` - Toggle version columns visibility
- `L` - Toggle log viewer
- `j/k` - Navigate between client logs (when logs visible)
- `g/G` - Jump to first/last client logs

## Default Ports

- **Prysm**: 3500
- **Lighthouse**: 5052
- **Teku**: 5051
- **Nimbus**: 5053

## Requirements

- Go 1.21 or later
- Access to Ethereum beacon node API endpoints
- Terminal with color support

## API Compatibility

watcheth uses the standard Ethereum Beacon API and connects to REST endpoints exposed by consensus, execution, and validator clients.

## Development

### Project Structure

```
watcheth/
├── cmd/              # CLI commands (monitor, list, debug, version)
├── internal/
│   ├── consensus/    # Consensus client implementations
│   ├── execution/    # Execution client implementations
│   ├── validator/    # Validator client support (Vouch)
│   ├── monitor/      # Display and monitoring logic
│   ├── config/       # Configuration management
│   └── common/       # Shared HTTP utilities
├── watcheth.yml      # Example configuration
└── main.go          # Entry point
```

### Building

Using the Makefile:

```bash
# Build for current platform
make build

# Build for Linux AMD64
make build-linux

# Build for all major platforms
make build-all

# Clean build artifacts
make clean

# Run tests
make test

# Install to GOPATH/bin
make install
```

Manual build (without Make):

```bash
go build -o watcheth
```

To build for Linux x86_64 systems (e.g., Ubuntu 64-bit servers):

```bash
GOOS=linux GOARCH=amd64 go build -o watcheth-linux-amd64
```

Build artifacts are placed in the `build/` directory with platform-specific subdirectories.

### Testing

```bash
go test ./...
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
