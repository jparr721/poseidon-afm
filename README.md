# Poseidon

<p align="center">
  <img alt="Poseidon Logo" src="documentation-payload/poseidon/poseidon.svg" height="30%" width="30%">
</p>

Poseidon is a cross-platform agent written in Go, targeting Windows, macOS, and Linux (x64 and ARM64).

This version uses a simple HTTP polling architecture with your own backend—no Mythic, no Docker, no complex C2 profiles.

## Quick Start

```bash
# Terminal 1: Start the test backend (requires uv)
cd poseidon/poseidon/agent_code
./test_backend.py

# Terminal 2: Run the agent
cd poseidon
go run main.go
```

The agent will check in and start polling for tasks on `http://localhost:11111`.

## Platform Support

The agent is platform-agnostic and works on:
- **Windows** (x64, arm64)
- **macOS** (x64, arm64)
- **Linux** (x64, arm64)

### Windows Notes

- Pure Go with no CGO requirements for the core polling client
- Some commands are stubs on Windows (JXA, XPC, Launchd, TCC, PTY, etc.)
- Elevation detection always reports non-elevated

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `POSEIDON_UI_BASEURL` | `http://localhost:11111` | Backend server URL |
| `POSEIDON_UI_CHECKIN_PATH` | `/checkin` | Check-in endpoint path |
| `POSEIDON_UI_POLL_PATH` | `/poll` | Polling endpoint path |
| `POSEIDON_UI_POLL_INTERVAL_SECONDS` | `5` | Seconds between polls |

## Architecture

```
poseidon/
├── main.go                     # Entry point - runs the polling client
└── poseidon/
    ├── agentfunctions/         # Command definitions (76 commands)
    └── agent_code/
        ├── test_backend.py     # FastAPI test server
        └── pkg/
            ├── ui/pollclient/  # HTTP polling client
            ├── tasks/          # Task processing
            ├── responses/      # Response aggregation
            └── utils/          # Structs, crypto, files
```

## HTTP Contract

Your backend needs to implement two endpoints:

### POST /checkin

Agent sends its info on startup.

**Request:** `structs.CheckInMessage`
```json
{
  "action": "checkin",
  "os": "Windows",
  "user": "username",
  "host": "hostname",
  "pid": 1234,
  "architecture": "amd64"
}
```

**Response:** Return an agent ID (or just 200 OK)
```json
{"id": "agent-123"}
```

### POST /poll

Agent polls for tasks and sends responses.

**Request:** `structs.MythicMessage`
```json
{
  "action": "get_tasking",
  "tasking_size": -1,
  "responses": []
}
```

**Response:** `structs.MythicMessageResponse`
```json
{
  "action": "get_tasking",
  "tasks": [
    {
      "id": "task-1",
      "command": "ls",
      "parameters": "{\"path\":\".\",\"depth\":1}",
      "timestamp": 1
    }
  ]
}
```

## Testing with test_backend.py

A ready-to-use FastAPI test server is included. Requires [uv](https://docs.astral.sh/uv/).

```bash
cd poseidon/poseidon/agent_code
./test_backend.py
```

### Curl Examples

**Check-in:**
```bash
curl -X POST http://localhost:11111/checkin \
  -H "Content-Type: application/json" \
  -d '{"action":"checkin","os":"Windows","user":"test","host":"testhost","pid":1234}'
```

**Poll for tasks:**
```bash
curl -X POST http://localhost:11111/poll \
  -H "Content-Type: application/json" \
  -H "X-Agent-ID: <agent-id>" \
  -d '{"action":"get_tasking","tasking_size":-1}'
```

**Queue a task:**
```bash
curl -X POST http://localhost:11111/queue_task \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"<agent-id>","command":"ls","parameters":"{\"path\":\".\",\"depth\":1}"}'
```

**List agents:**
```bash
curl http://localhost:11111/agents
```

## Icon Credit

Poseidon's icon made by Eucalyp from www.flaticon.com
