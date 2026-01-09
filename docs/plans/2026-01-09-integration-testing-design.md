# Integration Testing Framework Design

**Date:** 2026-01-09
**Status:** Approved

## Overview

Create a comprehensive integration testing framework for the Poseidon agent that tests commands end-to-end without requiring the full AFM-1 stack. The framework includes a mock AFM-1 API server, test harness, and per-command test definitions.

## Problem Statement

Currently there's no way to test the Poseidon agent end-to-end without running the full AFM-1 infrastructure. We need:

1. Smoke tests for basic commands (pwd, hostname, ls, shell)
2. A scalable framework to add all 76+ commands over time
3. Cross-platform support (Windows, macOS, Linux)
4. Simple execution via `go test`

## Architecture

```
poseidon/poseidon/agent_code/
├── cmd/
│   └── mockafm/                   # Standalone mock server (for manual testing)
│       └── main.go
│
├── pkg/
│   └── testing/
│       ├── README.md              # Documentation
│       ├── harness.go             # Orchestrates: build → spawn → test → cleanup
│       ├── mockafm/
│       │   ├── protocol.go        # AES-256-CBC + HMAC encryption
│       │   └── server.go          # HTTP server implementing AFM-1 /agent endpoint
│       └── commands/
│           ├── registry.go        # Command interface + test registry
│           ├── pwd.go             # pwd command test
│           ├── hostname.go        # hostname command test
│           ├── ls.go              # ls command test
│           └── shell.go           # shell command test
│
└── integration_test.go            # go test -tags=integration ./...
```

## Agent Execution Flow

The agent is built and executed the same way AFM-1 does it:

### Build
```bash
cd poseidon/poseidon/agent_code
go run ./cmd/builder --config config.json --output agent.bin
```

### Agent Entry Points
- `poseidon.go` - macOS/Linux (with CGO export for RunMain)
- `poseidon_windows.go` - Windows (no CGO)
- `poseidon_shared.go` - Shared library mode

All entry points initialize profiles, tasks, responses, files, p2p, then start C2 profiles.

### Execution
The built binary is spawned directly. It connects to the configured C2 endpoint (mock server in tests, AFM-1 API in production) via the HTTP profile.

## Protocol Specification

The mock server implements the AFM-1 agent protocol exactly.

### Encryption Format
```
base64(UUID[36 bytes] + IV[16 bytes] + AES-256-CBC(JSON) + HMAC-SHA256[32 bytes])
```

- **UUID**: 36-byte agent identifier (prepended to all messages)
- **IV**: 16-byte random initialization vector
- **Ciphertext**: AES-256-CBC encrypted JSON payload
- **HMAC**: SHA-256 HMAC over IV + Ciphertext for integrity

### Endpoint
```
POST /api/v1/operations/:operationId/agent
```

### Actions

| Action | Agent Sends | Server Returns |
|--------|-------------|----------------|
| `checkin` | `{action:"checkin", uuid, os, host, user, pid, ...}` | `{status:"success", id:"agent-db-id"}` |
| `get_tasking` | `{action:"get_tasking", responses:[...]}` | `{action:"get_tasking", tasks:[...]}` |

### Task Format (Server → Agent)
```json
{
  "id": "task-uuid",
  "command": "pwd",
  "parameters": "{}",
  "timestamp": 1704812345
}
```

### Response Format (Agent → Server)
```json
{
  "task_id": "task-uuid",
  "user_output": "/home/user",
  "completed": true,
  "status": "success"
}
```

## Test Harness Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Setup                                                       │
│     - Start mock AFM server on random available port            │
│     - Generate temp config JSON with server URL + test PSK      │
│     - Build agent: go run ./cmd/builder --config temp.json      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. Spawn Agent                                                 │
│     - exec.Command() the built binary                           │
│     - Wait for check-in (with 30s timeout)                      │
│     - Verify agent registered in mock server                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. Run Command Tests                                           │
│     - For each registered command test:                         │
│       a. Queue task in mock server                              │
│       b. Wait for agent to poll and respond (with timeout)      │
│       c. Run validator function on response                     │
│       d. Record pass/fail                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. Cleanup                                                     │
│     - Send "exit" command to agent                              │
│     - Kill process if still running after timeout               │
│     - Remove temp files (config, binary)                        │
│     - Stop mock server                                          │
└─────────────────────────────────────────────────────────────────┘
```

## Command Test Interface

### Registry (registry.go)
```go
type CommandTest struct {
    Name       string                      // Command name (e.g., "pwd")
    Parameters string                      // JSON parameters
    Validate   func(Response) error        // Validation function
    Setup      func(workdir string) error  // Optional: create test fixtures
    Teardown   func(workdir string) error  // Optional: cleanup test fixtures
}

type Response struct {
    TaskID      string
    UserOutput  string
    Completed   bool
    Status      string
    FileBrowser interface{}  // For ls command
    Processes   interface{}  // For ps command
    Stdout      string       // For shell command
    Stderr      string       // For shell command
}

func Register(test CommandTest)
func GetAll() []CommandTest
```

### Example: pwd.go
```go
package commands

import (
    "errors"
    "path/filepath"
    "strings"
)

func init() {
    Register(CommandTest{
        Name:       "pwd",
        Parameters: "{}",
        Validate: func(resp Response) error {
            output := strings.TrimSpace(resp.UserOutput)
            if output == "" {
                return errors.New("pwd returned empty output")
            }
            if !filepath.IsAbs(output) {
                return errors.New("pwd should return absolute path")
            }
            return nil
        },
    })
}
```

## Initial Command Tests

| Command | Parameters | Validation |
|---------|------------|------------|
| `pwd` | `{}` | Non-empty, absolute path |
| `hostname` | `{}` | Non-empty string |
| `ls` | `{"path": ".", "depth": 1}` | Has `file_browser` with files array |
| `shell` | `{"command": "echo hello"}` | `user_output` or `stdout` contains "hello" |

## Implementation Plan

### Phase 1: Mock Server
1. Create `pkg/testing/mockafm/protocol.go` - reuse `pkg/utils/crypto` for encryption
2. Create `pkg/testing/mockafm/server.go` - HTTP server with check-in and polling

### Phase 2: Command Framework
3. Create `pkg/testing/commands/registry.go` - command test interface and registry
4. Create `pkg/testing/commands/pwd.go`
5. Create `pkg/testing/commands/hostname.go`
6. Create `pkg/testing/commands/ls.go`
7. Create `pkg/testing/commands/shell.go`

### Phase 3: Test Harness
8. Create `pkg/testing/harness.go` - build, spawn, test, cleanup orchestration
9. Create `integration_test.go` - test entry point

### Phase 4: Documentation & Tooling
10. Create `pkg/testing/README.md` - usage documentation
11. Create `cmd/mockafm/main.go` - standalone server for manual testing

## Cross-Platform Considerations

- Use `os.MkdirTemp` for temp directories (cross-platform)
- Use `filepath` package for path manipulation (cross-platform separators)
- Use `exec.Command` for process spawning (cross-platform)
- No hardcoded paths - everything relative or configurable
- Binary extension handling: `.exe` on Windows, none on Unix
- Use `runtime.GOOS` for OS detection where needed

## Running Tests

```bash
# Run integration tests
cd poseidon/poseidon/agent_code
go test -tags=integration -v ./...

# Run with timeout (recommended)
go test -tags=integration -v -timeout=5m ./...

# Run specific command tests
go test -tags=integration -v -run=TestCommand/pwd ./...
```

## Future Enhancements

- Add more command tests incrementally (one file per command)
- Add parallel test execution for independent commands
- Add test coverage reporting
- Add CI/CD integration
- Add performance benchmarks for command execution
