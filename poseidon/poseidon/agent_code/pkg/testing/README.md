# Integration Testing Framework

This package provides tools for integration testing of the Poseidon agent against a mock AFM-1 server.

## Running Tests

Run integration tests with the `integration` build tag:

```bash
cd poseidon/poseidon/agent_code
go test -tags=integration -v ./...
```

Skip short mode tests with:
```bash
go test -tags=integration -v -short ./...
```

## Architecture

```
pkg/testing/
├── harness.go           # Test orchestration (build, spawn, test, cleanup)
├── mockafm/
│   ├── server.go        # Mock AFM-1 HTTP server
│   └── protocol.go      # Agent message encryption/decryption
└── commands/
    ├── registry.go      # Command test registration
    ├── pwd.go           # pwd command test
    ├── hostname.go      # hostname command test
    ├── ls.go            # ls command test
    └── shell.go         # shell command test
```

## Adding New Command Tests

Create a new file in `pkg/testing/commands/` following this pattern:

```go
package commands

import (
    "errors"
    "github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func init() {
    Register(CommandTest{
        Name:       "mycommand",
        Parameters: `{"arg1": "value1"}`,
        Validate: func(resp mockafm.Response) error {
            if resp.UserOutput == "" {
                return errors.New("expected output")
            }
            return nil
        },
        // Optional: Setup creates test fixtures
        Setup: func(workdir string) error {
            return nil
        },
        // Optional: Teardown cleans up fixtures
        Teardown: func(workdir string) error {
            return nil
        },
    })
}
```

Commands are automatically registered via `init()`.

## Manual Testing with Mock Server

Use the standalone mock server for interactive testing:

```bash
cd poseidon/poseidon/agent_code
go run ./cmd/mockafm -port 11111
```

Then build and run an agent configured to connect to `http://127.0.0.1:11111`.

## Test Harness API

```go
// Create harness with config
h := testing.NewHarness(testing.HarnessConfig{
    PSK:         "base64-encoded-32-byte-key",
    OperationID: "test-op",
    AgentUUID:   "uuid-string",
    BuildTags:   []string{"http"},
    Debug:       true,
})
defer h.Cleanup()

// Setup (starts server, builds agent)
h.Setup()

// Spawn agent process
h.SpawnAgent()

// Wait for agent check-in
h.WaitForCheckin(30 * time.Second)

// Run a command test
resp, err := h.RunCommand(cmd, 30 * time.Second)

// Access server directly for advanced testing
server := h.GetServer()
server.QueueTask(taskID, "pwd", "{}")
```

## Configuration

The harness generates a temporary config file and builds the agent using `cmd/builder`. Key config options:

- `PSK`: Base64-encoded 32-byte AES key
- `OperationID`: URL path component for agent endpoint
- `AgentUUID`: 36-character UUID
- `BuildTags`: Profiles to enable (e.g., `["http"]`)
- `BuildTimeout`: Agent build timeout (default: 2 minutes)
