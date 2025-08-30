# Debugging

## Basic Usage

```bash
# Auto-detect client type
watcheth debug http://localhost:5052

# Specify type explicitly
watcheth debug http://localhost:8545 --type execution
watcheth debug http://localhost:8081 --type vouch

# Save output
watcheth debug http://localhost:5052 --output results.txt

# Verbose mode
watcheth debug http://localhost:5052 -v
```

## What Gets Tested

**Consensus clients:**

- `/eth/v1/node/version`
- `/eth/v1/node/syncing`
- `/eth/v1/node/health`
- `/eth/v1/beacon/genesis`
- `/eth/v1/beacon/headers/head`
- `/eth/v1/node/peers`

**Execution clients:**

- `web3_clientVersion`
- `eth_syncing`
- `eth_blockNumber`
- `net_peerCount`

**Vouch:**

- Prometheus metrics at `/metrics`
