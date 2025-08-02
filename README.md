# WatchETH

A CLI tool for monitoring multiple Ethereum beacon nodes in real-time.

## Overview

WatchETH provides a terminal-based dashboard that displays real-time statistics from multiple Ethereum consensus clients (Prysm, Lighthouse, Teku, Nimbus). It shows important beacon chain information including:

- Current slot and head slot
- Current epoch and finality checkpoints
- Sync status and distance
- Time until next slot/epoch
- Connection status for each node

## Features

- **Multi-client Support**: Monitor Prysm, Lighthouse, Teku, and Nimbus nodes simultaneously
- **Real-time Updates**: Automatically refreshes beacon chain statistics
- **Terminal UI**: Clean, organized display using tview
- **Concurrent Monitoring**: Efficiently polls multiple nodes in parallel
- **Configuration**: Flexible YAML configuration for node endpoints

## Installation

```bash
go install github.com/watcheth/watcheth@latest
```

Or build from source:

```bash
git clone https://github.com/watcheth/watcheth
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

### Custom Configuration

Create a `watcheth.yaml` file:

```yaml
clients:
  - name: "Prysm Node 1"
    type: "prysm"
    endpoint: "http://localhost:3500"
  - name: "Lighthouse Node 1"
    type: "lighthouse"
    endpoint: "http://localhost:5052"
  - name: "Teku Node 1"
    type: "teku"
    endpoint: "http://localhost:5051"
  - name: "Nimbus Node 1"
    type: "nimbus"
    endpoint: "http://localhost:5053"

refresh_interval: 2s
```

Run with custom configuration:

```bash
watcheth --config /path/to/watcheth.yaml
```

### Display Modes

WatchETH now supports multiple display modes to fit different terminal sizes:

- **Compact View** (default) - Shows essential metrics that fit in 80 columns
- **Network View** - Focuses on network health metrics (peers, version, fork)
- **Consensus View** - Shows epoch and finalization information
- **Full View** - Displays all available metrics (requires wide terminal)

### Keyboard Shortcuts

- `q` - Quit the application
- `r` - Force refresh
- `1` - Switch to Compact view
- `2` - Switch to Network view  
- `3` - Switch to Consensus view
- `4` - Switch to Full view

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

WatchETH uses the standard Ethereum Beacon API specification, which is supported by all major consensus clients. The tool connects to the REST API endpoints exposed by each client.

## Development

### Project Structure

```
watcheth/
├── cmd/              # CLI commands
├── internal/
│   ├── beacon/       # Beacon API client
│   ├── monitor/      # Monitoring logic
│   └── config/       # Configuration
├── watcheth.yaml     # Default configuration
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
