# Installation

## Quick Install

```bash
go install github.com/attestantio/watcheth@latest
```

## Platform Binaries

Download from [releases](https://github.com/attestantio/watcheth/releases):

```bash
# Linux AMD64
wget https://github.com/attestantio/watcheth/releases/latest/download/watcheth-linux-amd64.tar.gz
tar -xzf watcheth-linux-amd64.tar.gz
sudo mv watcheth /usr/local/bin/

# macOS Apple Silicon
curl -L https://github.com/attestantio/watcheth/releases/latest/download/watcheth-darwin-arm64.tar.gz | tar -xz
sudo mv watcheth /usr/local/bin/
```

## Build from Source

```bash
git clone https://github.com/attestantio/watcheth
cd watcheth

# Simple build
go build

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o watcheth-linux
GOOS=darwin GOARCH=arm64 go build -o watcheth-mac

# Using Make
make build          # Current platform
make build-all      # All platforms
make install        # Install to GOPATH/bin
```

## Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o watcheth

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/watcheth /usr/local/bin/
CMD ["watcheth"]
```

```bash
docker build -t watcheth .
docker run -v $(pwd)/watcheth.yml:/watcheth.yml watcheth
```

## Verify Installation

```bash
watcheth version
watcheth --help
```
