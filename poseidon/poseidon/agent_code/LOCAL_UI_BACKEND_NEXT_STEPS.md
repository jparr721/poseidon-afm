# Backend & UI Next Steps (Polling Contract)

This document tells your backend team how to implement the poll contract, and your UI team how to consume it. The contract reuses Poseidon’s existing structs (`structs.MythicMessage` / `structs.MythicMessageResponse`) but does **not** depend on Mythic or Docker.

## Backend responsibilities

### Endpoints
- `POST /checkin`
  - Request: `structs.CheckInMessage`
  - Response: optional `{ "id": "<agent_id>" }` (200 OK is sufficient)
- `POST /poll`
  - Request: `structs.MythicMessage` (agent → backend; may include responses from the agent)
  - Response: `structs.MythicMessageResponse` (backend → agent; carries tasks and other queues)

### State you should keep
- Agent registry: map agent_id → last_seen, host/user info from `CheckInMessage`
- Task queue per agent: array of tasks (id, command, parameters, timestamp)
- Response sink per agent: store `responses` from the agent so the UI can fetch them

### Server-side flow (per `/poll`)
1. Parse the agent’s `MythicMessage`.
2. Persist any outbound data the agent sent:
   - `responses` (task outputs, file_browser, etc.)
   - `interactive`, `socks`, `rpfwd`, `delegates`, `edges` if you plan to support them later.
3. Pop queued tasks for this agent and return them in `MythicMessageResponse.tasks`.
4. Leave other fields empty unless you are using them (e.g., `interactive`, `delegates`).

### Example `/poll` response to send `ls`
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

### Handling agent responses
- The agent will POST `MythicMessage` with `responses` filled. Each `Response` has:
  - `task_id`, `user_output`, `completed`, `status`
  - Optional `file_browser`, `artifacts`, `stdout/stderr`, etc. (see `pkg/utils/structs/definitions.go`)
- Store these per-task so the UI can retrieve and render them.

## UI integration guidance

### How the UI should talk to your backend
- UI never talks directly to the agent. It only calls your backend.
- UI calls (examples):
  - `GET /agents` → list registered agents and their last_seen info.
  - `GET /agents/{id}/tasks` → list tasks and their statuses.
  - `GET /agents/{id}/responses?task_id=...` → fetch responses (including `file_browser` data).
  - `POST /agents/{id}/tasks` with `{command:"ls", parameters:"{\"path\":\".\",\"depth\":1}"}` to enqueue a task.

### Rendering `ls` results
- The `file_browser` object sits inside a `Response`:
  - `file_browser.files[]` contains entries with `name`, `full_name`, `is_file`, `permissions`, `size`, `modify_time`, `access_time`.
  - `file_browser.is_file` is false for directories; you can recurse if desired using returned `files`.
- Show `user_output` and `status` for quick feedback; use `file_browser` for structured rendering.

### Minimal UI flow
1. Operator selects an agent → UI shows its recent tasks/responses (from backend storage).
2. Operator queues a task (e.g., `ls`) → UI POSTs to backend to enqueue.
3. Backend returns that task on the next `/poll`; agent executes; backend stores the resulting `responses`.
4. UI polls backend (or uses WebSocket from backend) to show the new `file_browser` result.

## Notes and future extensions
- Keep the schema as-is for now; you can later rename fields for clarity without affecting the agent so long as you translate at the backend.
- If you later add push transports, you can reuse the same message shapes.

