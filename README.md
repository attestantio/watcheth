# WatchETH

A CLI tool for monitoring multiple Ethereum beacon nodes in real-time.

## Overview

WatchETH provides a terminal-based dashboard that displays real-time statistics from multiple Ethereum clients including consensus clients (Prysm, Lighthouse, Teku, Nimbus), execution clients, and validator clients (Vouch). It shows important information including:

- Current slot and head slot
- Current epoch and finality checkpoints
- Sync status and distance
- Time until next slot/epoch
- Connection status for each node

## Features

- **Multi-client Support**: Monitor consensus, execution, and validator clients simultaneously
  - Consensus: Prysm, Lighthouse, Teku, Nimbus
  - Execution: Geth, Nethermind, Besu, Erigon
  - Validator: Vouch (multi-node validator client)
- **Real-time Updates**: Automatically refreshes client statistics
- **Terminal UI**: Clean, organized display using tview
- **Concurrent Monitoring**: Efficiently polls multiple nodes in parallel
- **Configuration**: Flexible YAML configuration for node endpoints
- **Vouch Metrics**: Monitor validator performance including attestations, proposals, and sync committees

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

The debug command tests various API endpoints and shows their availability and response. For Vouch, it parses and displays key Prometheus metrics. Use `--output` or `-o` to save the results to a file while still displaying them on the terminal.

### Custom Configuration

Create a `watcheth.yaml` file:

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
watcheth --config /path/to/watcheth.yaml
```

### Display Modes

WatchETH now supports multiple display modes to fit different terminal sizes:

- **Compact View** (default) - Shows essential metrics that fit in 80 columns
- **Network View** - Focuses on network health metrics (peers, version, fork)
- **Consensus View** - Shows epoch and finalization information

### Keyboard Shortcuts

- `q` - Quit the application
- `r` - Force refresh
- `1` - Switch to Compact view
- `2` - Switch to Network view  
- `3` - Switch to Consensus view

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
