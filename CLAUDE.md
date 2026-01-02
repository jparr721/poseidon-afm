# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Poseidon is a Mythic 3.0+ C2 payload agent written in Go, targeting macOS and Linux (x86_64 and ARM64). It consists of two components:

1. **Payload Container** (`poseidon/`) - Interfaces with Mythic server via gRPC, defines 76+ commands
2. **Agent Runtime** (`poseidon/poseidon/agent_code/`) - The compiled agent that runs on target systems

Current version: 2.2.26

## Build Commands

### Container Build (Mythic integration)
```bash
cd poseidon
make build           # Full build with module downloads
make run             # Run with defaults
make run_custom      # Run with custom environment (DEBUG_LEVEL, RABBITMQ_HOST, MYTHIC_SERVER_HOST)
```

### Agent Build (standalone testing)
```bash
cd poseidon/poseidon/agent_code

# Single profile builds
make build_http          # HTTP/HTTPS beaconing
make build_websocket     # WebSocket connections
make build_tcp           # Direct TCP (P2P capable)
make build_dynamichttp   # HTTP with dynamic parameters
make build_httpx         # HTTP + XPC (macOS)
make build_dns           # DNS over gRPC (beta)

# Multi-profile builds
make build_http_tcp          # HTTP + TCP
make build_websocket_http    # WebSocket + HTTP

# Build and run
make build_and_run_http
make build_and_run_websocket

# Protocol buffer generation (DNS profile)
make build_protobuf_go
```

Edit the `test_agent_config_*.json` files to configure test builds. Copy UUID and AES key from Mythic Payloads page for standalone builds.

## Architecture

### Directory Structure
```
poseidon/
├── main.go                     # Entry point
├── go.mod                      # Container deps (Go 1.25.1)
├── Makefile
└── poseidon/
    ├── agentfunctions/         # Command definitions (76 commands)
    │   └── builder.go          # Payload build configuration
    ├── browserscripts/         # JavaScript UI rendering
    └── agent_code/             # Runtime agent
        ├── poseidon.go         # Agent entry point
        ├── go.mod              # Agent deps (Go 1.24.0)
        ├── Makefile
        └── pkg/
            ├── profiles/       # C2 communication implementations
            ├── tasks/          # Task processing and routing
            ├── responses/      # Response aggregation
            └── utils/          # Structs, crypto, file handling, P2P
```

### Command Registration Pattern

Each command is a separate file in `agentfunctions/` using `init()` for auto-registration:

```go
var myCommand = agentstructs.Command{
    Name: "mycommand",
    TaskFunctionCreateTasking: myCommandCreateTasking,
    MitreAttackMappings: []string{"T1059"},
}

func init() {
    agentstructs.AllPayloadData.Get("poseidon").AddCommand(myCommand)
}
```

### C2 Profile Interface

All profiles implement the interface in `pkg/profiles/profile.go`:
- **http** - Standard HTTP/HTTPS with proxy support
- **websocket** - Persistent connections (gorilla/websocket)
- **tcp** - Direct TCP, P2P capable
- **dynamichttp** - HTTP with dynamic parameter variation
- **httpx** - HTTP + macOS XPC integration
- **dns** - DNS over gRPC with TCP fallback (beta)
- **webshell** - HTTP-based P2P

Profiles support automatic failover based on `egress_order` and `failover_threshold` parameters.

### Platform-Specific Code

File suffix convention:
- `*_darwin.go` - macOS
- `*_linux.go` - Linux
- `*_darwin_amd64.go` / `*_darwin_arm64.go` - Architecture-specific

macOS features use Objective-C via CGO (XPC, clipboard, screenshots, prompts, keylogging).

### Agent Message Flow

1. C2 profile receives message and calls `tasks.HandleMessageFromMythic()`
2. Tasks tracked in running tasks map by `task_id`
3. Responses flow through channels in `pkg/responses/`
4. All responses aggregated and sent back via active profile

## Build Parameters (builder.go)

- **mode**: `default` | `c-archive` | `c-shared` (dylib/so output)
- **architecture**: `AMD_x64` | `ARM_x64`
- **garble**: Enable code obfuscation (slower builds)
- **debug**: Enable debug print statements
- **static**: Static compilation (Linux only)
- **egress_order**: Array of C2 profile priority
- **failover_threshold**: Failed attempts before profile rotation (default: 10)

## Installation

```bash
# Install via mythic-cli
sudo ./mythic-cli install github https://github.com/user/repo
sudo ./mythic-cli install github https://github.com/user/repo branchname
sudo ./mythic-cli install folder /path/to/local/folder
```

## Documentation

Rendered docs available in Mythic UI under **Docs -> Agent Documentation**.
Source in `documentation-payload/poseidon/` with command docs in `commands/` subdirectory.
