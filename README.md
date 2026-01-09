# Poseidon

<p align="center">
  <img alt="Poseidon Logo" src="documentation-payload/poseidon/poseidon.svg" height="30%" width="30%">
</p>

Poseidon is a Mythic 3.0+ C2 payload agent written in Go, targeting macOS and Linux (x86_64 and ARM64).

## Features

- **Multiple C2 Profiles**: HTTP, WebSocket, TCP, DNS, DynamicHTTP, HTTPx
- **Automatic Failover**: Configurable profile rotation on connection failures
- **76+ Commands**: File operations, process management, credential access, persistence, and more
- **Cross-Platform**: macOS and Linux with architecture-specific optimizations
- **Unified Config System**: JSON-driven builds with validation and dry-run support

## Quick Start

### Standalone Build (No Mythic)

```bash
cd poseidon/poseidon/agent_code

# Build with HTTP profile
make build_http

# Or use the builder directly
go run ./cmd/builder --config ./cmd/builder/testdata/http-test.json

# See all build targets
make help
```

### With Mythic

```bash
# Install the agent
sudo ./mythic-cli install github https://github.com/MythicAgents/poseidon

# Or from local folder
sudo ./mythic-cli install folder /path/to/poseidon
```

## C2 Profiles

| Profile | Description |
|---------|-------------|
| **http** | Standard HTTP/HTTPS beaconing with proxy support |
| **websocket** | Persistent WebSocket connections |
| **tcp** | Direct TCP, P2P capable |
| **dynamichttp** | HTTP with dynamic parameter variation |
| **httpx** | HTTP + macOS XPC integration |
| **dns** | DNS over gRPC (beta) |

Profiles support automatic failover based on `egress_order` and `failover_threshold` parameters.

## Build System

Poseidon uses a JSON-driven build system. Define your configuration in a JSON file:

```json
{
  "uuid": "agent-uuid",
  "debug": false,
  "build": {
    "os": "linux",
    "arch": "amd64",
    "output": "./agent.bin"
  },
  "profiles": ["http"],
  "egress": {
    "order": ["http"],
    "failover": "failover",
    "failedThreshold": 10
  },
  "http": {
    "callbackHost": "https://server.example.com",
    "callbackPort": 443,
    "aesPsk": "base64-encoded-key",
    "killdate": "2025-12-31",
    "interval": 10,
    "jitter": 20
  }
}
```

Then build:

```bash
# Validate config
go run ./cmd/builder --config config.json --validate

# Preview build
go run ./cmd/builder --config config.json --dry-run

# Build
go run ./cmd/builder --config config.json
```

Example configs are in `poseidon/poseidon/agent_code/cmd/builder/testdata/`.

## Architecture

```
poseidon/
├── main.go                     # Mythic container entry point
├── poseidon/
│   ├── agentfunctions/         # 76+ command definitions
│   │   └── builder.go          # Mythic build configuration
│   └── agent_code/             # Runtime agent
│       ├── poseidon.go         # Agent entry point
│       ├── Makefile            # Build targets
│       ├── cmd/builder/        # Unified config builder tool
│       └── pkg/
│           ├── config/         # Generated config package
│           ├── profiles/       # C2 profile implementations
│           ├── tasks/          # Task processing
│           ├── responses/      # Response aggregation
│           └── utils/          # Crypto, file handling, P2P
```

## Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS | x86_64, ARM64 | Full support |
| Linux | x86_64, ARM64 | Full support |

macOS-specific features (XPC, clipboard, screenshots, keylogging) use Objective-C via CGO.

## Build Parameters

| Parameter | Values | Description |
|-----------|--------|-------------|
| `mode` | `default`, `c-archive`, `c-shared` | Output format (dylib/so) |
| `architecture` | `AMD_x64`, `ARM_x64` | Target architecture |
| `garble` | `true/false` | Enable Garble obfuscation |
| `debug` | `true/false` | Enable debug output |
| `static` | `true/false` | Static compilation (Linux) |
| `egress_order` | `["http", "websocket"]` | C2 profile priority |
| `failover_threshold` | `10` | Failures before rotation |

## Documentation

Full documentation available in Mythic UI under **Docs -> Agent Documentation**.

Command documentation source: `documentation-payload/poseidon/commands/`

## Icon Credit

Poseidon's icon made by Eucalyp from www.flaticon.com
