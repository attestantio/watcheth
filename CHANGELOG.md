# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-08-29

### Added
- Initial release of watcheth CLI tool
- Multi-client monitoring support for Ethereum consensus clients:
  - Prysm
  - Lighthouse
  - Teku
  - Nimbus
- Execution client monitoring support for:
  - Geth
  - Nethermind
  - Besu
  - Erigon
- Validator monitoring with Vouch integration
- Real-time log monitoring and display
- Interactive terminal UI with keyboard shortcuts:
  - `Tab` / `Shift+Tab` - Navigate between sections
  - `Enter` - Toggle log following
  - `Space` - Select current node (in list view)
  - `↑`/`↓` - Navigate nodes
  - `←`/`→` - Navigate sections
  - `q` / `Ctrl+C` - Quit application
- YAML configuration support
- Debug mode for troubleshooting connectivity issues
- Version command with build information
- Cross-platform support (Linux, macOS, Windows)
- Prometheus metrics parsing for node monitoring

[0.1.0]: https://github.com/watcheth/watcheth/releases/tag/v0.1.0