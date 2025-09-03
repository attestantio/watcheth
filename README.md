# watcheth

Unified monitoring for Ethereum consensus, execution, and validator clients.

## Features

- Monitor consensus clients (Prysm, Lighthouse, etc.), execution clients (Geth, Nethermind, etc.), and Vouch
- Real-time updates with live log streaming
- Clean terminal UI with keyboard navigation
- Debug mode for troubleshooting endpoints

## Quick Start

```bash
# Install
go install github.com/attestantio/watcheth@latest

# Run
watcheth
```

## Configuration

```yaml
# watcheth.yml
clients:
  - name: "Prysm"
    type: consensus
    endpoint: "http://localhost:3500"
  - name: "Geth"
    type: execution
    endpoint: "http://localhost:8545"

refresh_interval: 2s
```

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Usage](docs/navigation.md)
- [Debugging](docs/debugging.md)
- [Contributing](CONTRIBUTING.md)

## License

Apache License 2.0
