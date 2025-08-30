# Configuration

## Basic Structure

```yaml
clients:
  - name: "Display name"
    type: consensus | execution | vouch
    endpoint: "http://localhost:PORT"

refresh_interval: 2s # Default: 2s
```

## Examples

### Simple Setup

```yaml
clients:
  - name: "Prysm"
    type: consensus
    endpoint: "http://localhost:3500"
  - name: "Geth"
    type: execution
    endpoint: "http://localhost:8545"
```

### Multiple Nodes

```yaml
clients:
  - name: "Prysm Primary"
    type: consensus
    endpoint: "http://node1:3500"
  - name: "Prysm Backup"
    type: consensus
    endpoint: "http://node2:3500"
```

### Remote Monitoring

```yaml
clients:
  - name: "Remote Beacon"
    type: consensus
    endpoint: "https://beacon.example.com"
refresh_interval: 5s # Higher for remote
```

## Environment Variables

```bash
export WATCHETH_REFRESH_INTERVAL=5s
```

## Debugging

Test endpoints before monitoring:

```bash
watcheth debug http://localhost:5052
watcheth debug http://localhost:8545 --type execution
```
