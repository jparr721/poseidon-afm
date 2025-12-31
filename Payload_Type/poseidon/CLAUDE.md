# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Poseidon is a Mythic C2 payload agent written in Go, targeting macOS and Linux. It consists of two main components:
1. **Payload Container** (root level) - Interfaces with Mythic server via gRPC
2. **Agent Code** (`poseidon/agent_code/`) - The actual agent that runs on target systems

Current version: 2.2.25

## Build Commands

```bash
# Full build (container context)
make build

# Run with defaults
make run

# Run with custom configuration
make run_custom \
  DEBUG_LEVEL=debug \
  RABBITMQ_HOST=192.168.1.100 \
  MYTHIC_SERVER_HOST=192.168.1.100
```

The Dockerfile uses `itsafeaturemythic/mythic_go_macos:latest` as the base image with Garble for code obfuscation.

## Architecture

### Directory Structure

```
poseidon/
├── main.go                 # Entry point - initializes commands and starts Mythic container
├── go.mod                  # Container dependencies (Go 1.25.1)
├── poseidon/
│   ├── agentfunctions/     # Command definitions (76 commands)
│   │   └── builder.go      # Build system and payload configuration
│   └── agent_code/         # Runtime agent
│       ├── poseidon.go     # Agent entry point
│       ├── go.mod          # Agent dependencies (Go 1.24.0)
│       └── pkg/
│           ├── profiles/   # C2 communication (http, websocket, tcp, dns, etc.)
│           ├── tasks/      # Task processing and routing
│           ├── responses/  # Response aggregation
│           └── utils/      # Structs, crypto, file handling
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

### C2 Profiles

Profiles implement a common interface in `pkg/profiles/profile.go`. Available profiles:
- **http** - Standard HTTP/HTTPS beaconing
- **websocket** - Persistent WebSocket connections
- **tcp** - Direct TCP (P2P capable)
- **dynamichttp** - HTTP with dynamic parameter variation
- **httpx** - HTTP with XPC integration (macOS)
- **dns** - DNS over GRPC (beta)
- **webshell** - HTTP-based P2P

Profiles support automatic failover based on `egress_order` and `failover_threshold` build parameters.

### Platform-Specific Code

Use build tags and file suffixes for platform-specific implementations:
- `*_darwin.go` - macOS
- `*_linux.go` - Linux
- `*_darwin_amd64.go` / `*_darwin_arm64.go` - Architecture-specific

### Agent Message Flow

1. Profiles call `tasks.HandleMessageFromMythic()` on received messages
2. Tasks are tracked in a running tasks map by `task_id`
3. Responses flow through channels in `pkg/responses/`
4. All responses are aggregated and sent back via the active profile

## Build Parameters (builder.go)

Key build-time parameters:
- **mode**: `default` | `c-archive` | `c-shared` (dylib/so output)
- **architecture**: `AMD_x64` | `ARM_x64`
- **garble**: Enable Garble obfuscation (slower builds)
- **debug**: Enable debug print statements
- **static**: Static compilation (Linux only)
- **egress_order**: Array of C2 profile priority
- **failover_threshold**: Failed attempts before profile rotation (default: 10)

## Key Dependencies

Container:
- `github.com/MythicMeta/MythicContainer` v1.6.3

Agent:
- `github.com/gorilla/websocket` - WebSocket profile
- `github.com/creack/pty` - PTY handling
- `golang.org/x/crypto` - SSH, encryption
- `howett.net/plist` - macOS plist parsing
